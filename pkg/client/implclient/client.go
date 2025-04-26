package implclient

import (
	"context"
	"crypto/tls"
	"github.com/Li-giegie/node/internal"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/conn/implconn"
	"github.com/Li-giegie/node/pkg/errors"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"net"
	"strconv"
	"sync"
	"time"
)

type Client struct {
	Id uint32
	// 远程节点ID
	RemoteID uint32
	// 认证密钥
	RemoteKey []byte
	// 认证超时时长
	AuthTimeout time.Duration
	// 大于1时启用，并发请求或发送时，发出的消息不会被立即发出，而是会进入队列，直至队列缓冲区满，或者最后一个goroutine时才会将消息发出，如果消息要以最快的方式发出，那么请不要进入队列
	WriterQueueSize int
	// 读缓存区大小
	ReaderBufSize int
	// 大于64时启用，从队列读取后进入缓冲区，缓冲区大小
	WriterBufSize int
	OnCloseFunc   func(err error)
	internalField
}

type internalField struct {
	recvChan map[uint32]chan *message.Message
	recvLock sync.Mutex
	state    bool
	stopCtx  context.Context
	cancel   context.CancelFunc
	*implconn.Conn
	keepaliveInterval     time.Duration
	keepaliveTimeout      time.Duration
	keepaliveTimeoutClose time.Duration
	handle                map[uint8]func(w responsewriter.ResponseWriter, m *message.Message)
}

// Connect address 支持url格式例如 tcp://127.0.0.1:5555 = 127.0.0.1:5555，缺省协议默认tcp，config参数只能接受0个或者1个
func (c *Client) Connect(address string, config ...*tls.Config) (err error) {
	network, addr := internal.ParseAddr(address)
	var native net.Conn
	if n := len(config); n > 0 {
		if n != 1 {
			panic("only one config option is allowed")
		}
		native, err = tls.Dial(network, addr, config[0])
	} else {
		native, err = net.Dial(network, addr)
	}
	if err != nil {
		return err
	}
	return c.Start(native)
}

func (c *Client) Start(native net.Conn) (err error) {
	defer func() {
		if err != nil {
			_ = native.Close()
		}
	}()
	err = internal.DefaultAuthService.Request(native, &internal.BaseAuthRequest{
		ConnType: conn.NodeTypeClient,
		SrcId:    c.Id,
		DstId:    c.RemoteID,
		Key:      c.RemoteKey,
	})
	if err != nil {
		return err
	}
	resp, err := internal.DefaultAuthService.ReadResponse(native, c.AuthTimeout)
	if err != nil {
		return err
	}
	if resp.Code != internal.BaseAuthResponseCodeSuccess {
		return errors.New(resp.Code.String())
	}
	c.recvChan = make(map[uint32]chan *message.Message)
	c.Conn = implconn.NewConn(resp.ConnType, c.Id, c.RemoteID, native, c.recvChan, &c.recvLock, new(uint32), c.ReaderBufSize, c.WriterBufSize, c.WriterQueueSize, resp.MaxMsgLen)
	c.keepaliveInterval = resp.KeepaliveTimeout / 2
	c.keepaliveTimeout = resp.KeepaliveTimeout / 2
	c.keepaliveTimeoutClose = resp.KeepaliveTimeoutClose
	go c.Serve()
	return nil
}

func (c *Client) Serve() (err error) {

	errChan := make(chan error, 1)
	c.stopCtx, c.cancel = context.WithCancel(context.Background())
	go c.StartKeepalive()
	c.state = true
	defer func() {
		c.state = false
		if c.OnCloseFunc != nil {
			c.OnCloseFunc(err)
		}
		_ = c.Conn.Close()
	}()
	go func() {
		for {
			msg, err := c.Conn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			msg.Hop++
			if msg.DestId != c.Id {
				continue
			}
			switch msg.Type {
			case message.MsgType_KeepaliveASK:
				_ = c.Conn.SendType(message.MsgType_KeepaliveACK, nil)
			case message.MsgType_KeepaliveACK:
			case message.MsgType_Response:
				c.recvLock.Lock()
				ch, ok := c.recvChan[msg.Id]
				if ok {
					ch <- msg
					delete(c.recvChan, msg.Id)
				}
				c.recvLock.Unlock()
			default:
				h, ok := c.handle[msg.Type]
				if ok {
					h(&internal.ResponseWriter{
						Conn:     c.Conn,
						MsgId:    msg.Id,
						MsgDstId: msg.SrcId,
					}, msg)
				}
			}
		}
	}()
	select {
	case err = <-errChan:
		return err
	case <-c.stopCtx.Done():
		return nil
	}
}

func (c *Client) Register(typ uint8, handler func(w responsewriter.ResponseWriter, m *message.Message)) {
	if c.handle == nil {
		c.handle = make(map[uint8]func(w responsewriter.ResponseWriter, m *message.Message))
	}
	if c.handle[typ] != nil {
		panic("register type exists" + strconv.Itoa(int(typ)))
	}
	c.handle[typ] = handler
}

func (c *Client) OnClose(f func(err error)) {
	c.OnCloseFunc = f
}

func (c *Client) OnMessage(f func(w responsewriter.ResponseWriter, m *message.Message)) {
	c.Register(message.MsgType_Default, f)
}

func (c *Client) NodeId() uint32 {
	return c.Id
}

func (c *Client) Close() error {
	if c.Conn == nil {
		return nil
	}
	if c.cancel != nil {
		c.cancel()
	}
	return c.Conn.Close()
}

func (c *Client) StartKeepalive() {
	if c.keepaliveInterval < time.Millisecond*100 {
		c.keepaliveInterval = time.Millisecond * 100
	}
	tick := time.NewTicker(c.keepaliveInterval)
	defer tick.Stop()
	go func() {
		<-c.stopCtx.Done()
		tick.Stop()
	}()
	var diff int64
	var err error
	for t := range tick.C {
		diff = t.UnixNano() - int64(c.Conn.Activate())
		if diff >= int64(c.keepaliveTimeoutClose) {
			_ = c.Conn.Close()
		} else if diff >= int64(c.keepaliveTimeout) {
			if err = c.Conn.SendType(message.MsgType_KeepaliveASK, nil); err != nil {
				_ = c.Conn.Close()
			}
		}
	}
}

func (c *Client) State() bool {
	return c.state
}

package client

import (
	"context"
	"crypto/tls"
	"github.com/Li-giegie/node/internal"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/errors"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/reply"
	"net"
	"sync"
	"sync/atomic"
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
	internalField
}
type State uint32

const (
	StateClosed State = iota
	StateRunning
)

type internalField struct {
	recvChan map[uint32]chan *message.Message
	recvLock sync.Mutex
	state    State
	*conn.Conn
	keepaliveInterval     time.Duration
	keepaliveTimeout      time.Duration
	keepaliveTimeoutClose time.Duration
	Handler
}

// Connect address 支持url格式例如 tcp://127.0.0.1:5555 = 127.0.0.1:5555，缺省协议默认tcp，config参数只能接受0个或者1个
func (c *Client) Connect(address string, h Handler, config ...*tls.Config) (err error) {
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
	return c.Start(native, h)
}

func (c *Client) Start(native net.Conn, h Handler) (err error) {
	defer func() {
		if err != nil {
			_ = native.Close()
		}
	}()
	if h != nil {
		c.Handler = h
	} else {
		c.Handler = &Default
	}
	err = internal.DefaultAuthService.Request(native, &internal.BaseAuthRequest{
		ConnType: conn.TypeClient,
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
	c.Conn = conn.NewConn(resp.ConnType, c.Id, c.RemoteID, native, c.recvChan, &c.recvLock, new(uint32), c.ReaderBufSize, c.WriterBufSize, c.WriterQueueSize, resp.MaxMsgLen)
	c.keepaliveInterval = resp.KeepaliveTimeout / 2
	c.keepaliveTimeout = resp.KeepaliveTimeout / 2
	c.keepaliveTimeoutClose = resp.KeepaliveTimeoutClose
	go c.Serve()
	return nil
}

func (c *Client) Serve() (err error) {
	c.state = StateRunning
	ctx, cancel := context.WithCancel(context.Background())
	go c.StartKeepalive(ctx)
	defer func() {
		c.state = StateClosed
		cancel()
		_ = c.Conn.Close()
		c.Handler.OnClose(c.Conn, err)
	}()
	for {
		msg, err := c.Conn.ReadMessage()
		if err != nil {
			if c.state == StateRunning {
				return err
			}
			return nil
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
			c.Handler.OnMessage(reply.NewReply(c.Conn, msg.Id, msg.SrcId), msg)
		}
	}
}

func (c *Client) NodeId() uint32 {
	return c.Id
}

func (c *Client) Close() error {
	if atomic.CompareAndSwapUint32((*uint32)(&c.state), uint32(StateRunning), uint32(StateClosed)) {
		return c.Conn.Close()
	}
	return nil
}

func (c *Client) StartKeepalive(ctx context.Context) {
	if c.keepaliveInterval < time.Millisecond*100 {
		c.keepaliveInterval = time.Millisecond * 100
	}
	tick := time.NewTicker(c.keepaliveInterval)
	defer tick.Stop()
	go func() {
		<-ctx.Done()
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

func (c *Client) State() State {
	return c.state
}

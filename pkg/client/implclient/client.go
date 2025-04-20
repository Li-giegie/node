package implclient

import (
	"context"
	"crypto/tls"
	"github.com/Li-giegie/node/internal"
	"github.com/Li-giegie/node/internal/handlermanager"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/conn/implconn"
	"github.com/Li-giegie/node/pkg/errors"
	"github.com/Li-giegie/node/pkg/message"
	"net"
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
	// 大于0时启用，收发消息最大长度，最大值0xffffffff
	MaxMsgLen uint32
	// 大于1时启用，并发请求或发送时，发出的消息不会被立即发出，而是会进入队列，直至队列缓冲区满，或者最后一个goroutine时才会将消息发出，如果消息要以最快的方式发出，那么请不要进入队列
	WriterQueueSize int
	// 读缓存区大小
	ReaderBufSize int
	// 大于64时启用，从队列读取后进入缓冲区，缓冲区大小
	WriterBufSize int
	// 连接保活检查时间间隔
	KeepaliveInterval time.Duration
	// 连接保活超时时间
	KeepaliveTimeout time.Duration
	// 连接保活最大超时次数
	KeepaliveTimeoutClose time.Duration
	internalField
}

type internalField struct {
	recvChan map[uint32]chan *message.Message
	recvLock sync.Mutex
	state    bool
	stopCtx  context.Context
	cancel   context.CancelFunc
	handlemanager.HandlerManager
	*implconn.Conn
}

// Connect address 支持url格式例如 tcp://127.0.0.1:5555 = 127.0.0.1:5555，缺省协议默认tcp，config参数只能接受0个或者1个
func (c *Client) Connect(address string, config ...*tls.Config) (err error) {
	network, addr := internal.ParseAddr(address)
	var native net.Conn
	if n := len(config); n > 0 {
		if n != 1 {
			return errors.MultipleConfigErr
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
	if !c.CallOnAccept(native) {
		return errors.AcceptDeniedErr
	}
	var dstType conn.NodeType
	if dstType, err = internal.Auth(native, conn.NodeTypeClient, c.Id, c.RemoteID, c.RemoteKey, c.AuthTimeout); err != nil {
		return err
	}
	c.stopCtx, c.cancel = context.WithCancel(context.Background())
	c.recvChan = make(map[uint32]chan *message.Message)
	c.Conn = implconn.NewConn(dstType, c.Id, c.RemoteID, native, c.recvChan, &c.recvLock, new(uint32), c.ReaderBufSize, c.WriterBufSize, c.WriterQueueSize, c.MaxMsgLen)
	go c.Serve()
	return nil
}

func (c *Client) Serve() (err error) {
	c.startHeartbeatCheck()
	c.CallOnConnect(c.Conn)
	errChan := make(chan error, 1)
	c.state = true
	defer func() {
		c.state = false
		_ = c.Conn.Close()
		c.CallOnClose(c.Conn, err)
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
				c.CallOnMessage(&internal.ResponseWriter{
					Conn:     c.Conn,
					MsgId:    msg.Id,
					MsgDstId: msg.SrcId,
				}, msg)
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

func (c *Client) startHeartbeatCheck() {
	go func() {
		tick := time.NewTicker(c.KeepaliveInterval)
		defer tick.Stop()
		go func() {
			<-c.stopCtx.Done()
			tick.Stop()
		}()
		var diff int64
		var err error
		for t := range tick.C {
			diff = t.UnixNano() - int64(c.Conn.Activate())
			if diff >= int64(c.KeepaliveTimeoutClose) {
				_ = c.Conn.Close()
			} else if diff >= int64(c.KeepaliveTimeout) {
				if err = c.Conn.SendType(message.MsgType_KeepaliveASK, nil); err != nil {
					_ = c.Conn.Close()
				}
			}
		}
	}()
	return
}

func (c *Client) State() bool {
	return c.state
}

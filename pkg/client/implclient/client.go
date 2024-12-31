package implclient

import (
	"context"
	"crypto/tls"
	"github.com/Li-giegie/node/internal"
	"github.com/Li-giegie/node/internal/eventmanager/impleventmanager"
	"github.com/Li-giegie/node/pkg/conn/implconn"
	"github.com/Li-giegie/node/pkg/ctx/implcontext"
	"github.com/Li-giegie/node/pkg/errors"
	"github.com/Li-giegie/node/pkg/message"
	"net"
	"sync"
	"time"
)

type Client struct {
	Id          uint32
	Rid         uint32
	AuthKey     []byte
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
	RecvChan              map[uint32]chan *message.Message
	RecvLock              sync.Mutex
	state                 bool
	stopCtx               context.Context
	cancel                context.CancelFunc
	*impleventmanager.EventManager
	*implconn.Conn
}

// Connect address 支持url格式例如 tcp://127.0.0.1:5555 = 127.0.0.1:5555，缺省协议默认tcp，config参数只能接受0个或者1个
func (c *Client) Connect(address string, config ...*tls.Config) (err error) {
	network, addr := internal.ParseAddr(address)
	var conn net.Conn
	if n := len(config); n > 0 {
		if n != 1 {
			return errors.MultipleConfigErr
		}
		conn, err = tls.Dial(network, addr, config[0])
	} else {
		conn, err = net.Dial(network, addr)
	}
	if err != nil {
		return err
	}
	return c.Start(conn)
}

func (c *Client) Start(conn net.Conn) (err error) {
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	c.stopCtx, c.cancel = context.WithCancel(context.Background())
	if !c.CallOnAccept(conn) {
		return errors.AcceptDeniedErr
	}
	if err = internal.DefaultBasicReq.Send(conn, c.Id, c.Rid, c.AuthKey); err != nil {
		return err
	}
	permit, msg, err := internal.DefaultBasicResp.Receive(conn, c.AuthTimeout)
	if err != nil {
		return err
	}
	c.Conn = implconn.NewConn(c.Id, c.Rid, conn, c.RecvChan, &c.RecvLock, new(uint32), c.ReaderBufSize, c.WriterBufSize, c.WriterQueueSize, c.MaxMsgLen)
	if !permit {
		return errors.Error(msg)
	}
	c.startHeartbeatCheck()
	c.CallOnConnect(c.Conn)
	go func() {
		err = c.serve()
		c.CallOnClose(c.Conn, err)
	}()
	return nil
}

func (c *Client) serve() error {
	errChan := make(chan error, 1)
	c.state = true
	defer func() {
		_ = c.Conn.Close()
		c.state = false
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
			case message.MsgType_Reply:
				c.RecvLock.Lock()
				ch, ok := c.RecvChan[msg.Id]
				if ok {
					ch <- msg
					delete(c.RecvChan, msg.Id)
				}
				c.RecvLock.Unlock()
			default:
				c.CallOnMessage(implcontext.NewContext(c.Conn, msg))
			}
		}
	}()
	select {
	case err := <-errChan:
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
	c.cancel()
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

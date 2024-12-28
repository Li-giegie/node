package impl_client

import (
	"context"
	"crypto/tls"
	"github.com/Li-giegie/node/internal"
	"github.com/Li-giegie/node/internal/eventhandlerregistry/impl_eventhandlerregistry"
	"github.com/Li-giegie/node/pkg/common"
	"github.com/Li-giegie/node/pkg/conn/impl_conn"
	"github.com/Li-giegie/node/pkg/ctx/impl_context"
	"github.com/Li-giegie/node/pkg/errors/impl_errors"
	"github.com/Li-giegie/node/pkg/message"
	"net"
	"sync"
	"time"
)

type Client struct {
	id          uint32
	remoteId    uint32
	remoteKey   []byte
	authTimeout time.Duration
	recvChan    map[uint32]chan *message.Message
	recvLock    sync.Mutex
	stopCtx     context.Context
	cancel      context.CancelFunc
	*impl_conn.Conn
	*impl_eventhandlerregistry.EventHandlerRegistry
	*common.Config
}

// NewClient 创建一个客户端，remote字段为对端信息,conf为nil使用默认配置
func NewClient(localId uint32, remote *common.Identity, conf ...*common.Config) *Client {
	c := new(Client)
	c.id = localId
	c.remoteId = remote.Id
	c.remoteKey = remote.Key
	c.authTimeout = remote.AuthTimeout
	c.EventHandlerRegistry = impl_eventhandlerregistry.NewEventHandlerRegistry()
	c.recvChan = make(map[uint32]chan *message.Message)
	c.stopCtx, c.cancel = context.WithCancel(context.Background())
	if n := len(conf); n > 0 {
		if n != 1 {
			panic(impl_errors.MultipleConfigErr)
		}
		c.Config = conf[0]
	} else {
		c.Config = common.DefaultConfig
	}
	return c
}

// Connect address 支持url格式例如 tcp://127.0.0.1:5555 = 127.0.0.1:5555，缺省协议默认tcp，config参数只能接受0个或者1个
func (c *Client) Connect(address string, config ...*tls.Config) (err error) {
	network, addr := internal.ParseAddr(address)
	var conn net.Conn
	if n := len(config); n > 0 {
		if n != 1 {
			return impl_errors.MultipleConfigErr
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

func (c *Client) Start(conn net.Conn) error {
	if !c.CallOnAccept(conn) {
		return impl_errors.AcceptDeniedErr
	}
	if err := internal.DefaultBasicReq.Send(conn, c.id, c.remoteId, c.remoteKey); err != nil {
		return err
	}
	permit, msg, err := internal.DefaultBasicResp.Receive(conn, c.authTimeout)
	if err != nil {
		return err
	}
	c.Conn = impl_conn.NewConn(c.id, c.remoteId, conn, c.recvChan, &c.recvLock, new(uint32), c.ReaderBufSize, c.WriterBufSize, c.WriterQueueSize, c.MaxMsgLen)
	if !permit {
		return impl_errors.NodeError(msg)
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
	defer c.Conn.Close()
	go func() {
		for {
			msg, err := c.Conn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			msg.Hop++
			if msg.DestId != c.id {
				continue
			}
			switch msg.Type {
			case message.MsgType_KeepaliveASK:
				_ = c.Conn.SendType(message.MsgType_KeepaliveACK, nil)
			case message.MsgType_KeepaliveACK:
			case message.MsgType_Reply:
				c.recvLock.Lock()
				ch, ok := c.recvChan[msg.Id]
				if ok {
					ch <- msg
					delete(c.recvChan, msg.Id)
				}
				c.recvLock.Unlock()
			default:
				c.CallOnMessage(impl_context.NewContext(c.Conn, msg))
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

func (c *Client) Id() uint32 {
	return c.id
}

func (c *Client) Close() error {
	c.cancel()
	if c.Conn == nil {
		return nil
	}
	return c.Conn.Close()
}

func (c *Client) SetKeepalive(interval, timeout, timeoutClose time.Duration) {
	c.KeepaliveInterval = interval
	c.KeepaliveTimeout = timeout
	c.KeepaliveTimeoutClose = timeoutClose
	if !c.Keepalive {
		c.Keepalive = true
		c.startHeartbeatCheck()
	}
}

func (c *Client) startHeartbeatCheck() {
	if !c.Keepalive {
		return
	}
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

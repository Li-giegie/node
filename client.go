package node

import (
	"crypto/tls"
	"errors"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	nodeNet "github.com/Li-giegie/node/net"
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
	*nodeNet.Conn
	*nodeNet.ConnectionLifecycle
	*Config
}

// NewClient 创建一个客户端，remote字段为对端信息,conf为nil使用默认配置
func NewClient(localId uint32, remote *Identity, conf ...*Config) iface.Client {
	c := new(Client)
	c.id = localId
	c.remoteId = remote.Id
	c.remoteKey = remote.Key
	c.authTimeout = remote.AuthTimeout
	c.ConnectionLifecycle = nodeNet.NewConnectionLifecycle()
	c.recvChan = make(map[uint32]chan *message.Message)
	if n := len(conf); n > 0 {
		if n != 1 {
			panic("config accepts only one parameter")
		}
		c.Config = conf[0]
	} else {
		c.Config = DefaultConfig
	}
	return c
}

// Connect address 支持url格式例如 tcp://127.0.0.1:5555 = 127.0.0.1:5555，缺省协议默认tcp，config参数只能接受0个或者1个
func (c *Client) Connect(address string, config ...*tls.Config) (err error) {
	network, addr := parseAddr(address)
	var conn net.Conn
	if n := len(config); n > 0 {
		if n != 1 {
			panic("config accepts only one parameter")
		}
		conn, err = tls.Dial(network, addr, config[0])
	} else {
		conn, err = net.Dial(network, addr)
	}
	if err != nil {
		return err
	}
	if !c.CallOnAccept(conn) {
		return errors.New("AcceptCallback denied the connection establishment")
	}
	err = c.authenticate(conn)
	if err != nil {
		return err
	}
	go c.serve()
	return nil
}

func (c *Client) Start(conn net.Conn) error {
	if !c.CallOnAccept(conn) {
		return errors.New("AcceptCallback denied the connection establishment")
	}
	err := c.authenticate(conn)
	if err != nil {
		return err
	}
	go c.serve()
	return nil
}

func (c *Client) authenticate(conn net.Conn) (err error) {
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	err = nodeNet.DefaultBasicReq.Send(conn, c.id, c.remoteId, c.remoteKey)
	if err != nil {
		return err
	}
	permit, msg, err := nodeNet.DefaultBasicResp.Receive(conn, c.authTimeout)
	if err != nil {
		return err
	}
	node := nodeNet.NewConn(c.id, c.remoteId, conn, c.recvChan, &c.recvLock, new(uint32), c.ReaderBufSize, c.WriterBufSize, c.WriterQueueSize, c.MaxMsgLen)
	if !permit {
		return errors.New(msg)
	}
	c.Conn = node
	return nil
}

func (c *Client) serve() {
	c.CallOnConnect(c.Conn)
	for {
		msg, err := c.Conn.ReadMessage()
		if err != nil {
			_ = c.Conn.Close()
			c.CallOnClose(c.Conn, err)
			return
		}
		msg.Hop++
		if msg.DestId != c.id {
			//c.CallOnRouteMessage(nodeNet.NewContext(c.Conn, msg))
			continue
		}
		if msg.Type == message.MsgType_Reply {
			c.recvLock.Lock()
			ch, ok := c.recvChan[msg.Id]
			if ok {
				ch <- msg
				delete(c.recvChan, msg.Id)
			}
			c.recvLock.Unlock()
		} else {
			c.CallOnMessage(nodeNet.NewContext(c.Conn, msg))
		}
	}
}

func (c *Client) Id() uint32 {
	return c.id
}

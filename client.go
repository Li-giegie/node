package node

import (
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
	eventManager
	*Config
}

// NewClient 创建一个客户端，remote字段为对端信息,conf为nil使用默认配置
func NewClient(localId uint32, remote *Identity, conf *Config) iface.Client {
	c := new(Client)
	c.id = localId
	c.remoteId = remote.Id
	c.remoteKey = remote.Key
	c.authTimeout = remote.Timeout
	c.recvChan = make(map[uint32]chan *message.Message)
	c.Config = conf
	if c.Config == nil {
		c.Config = defaultConfig
	}
	return c
}

func (c *Client) Start(conn net.Conn) (iface.Conn, error) {
	node, err := c.authenticate(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	go c.serve(node)
	return node, nil
}

func (c *Client) authenticate(conn net.Conn) (*nodeNet.Connect, error) {
	err := defaultBasicReq.Send(conn, c.id, c.remoteId, c.remoteKey)
	if err != nil {
		return nil, err
	}
	permit, msg, err := defaultBasicResp.Receive(conn, c.authTimeout)
	if err != nil {
		return nil, err
	}
	node := nodeNet.NewConn(c.id, c.remoteId, conn, c.recvChan, &c.recvLock, new(uint32), c.ReaderBufSize, c.WriterBufSize, c.WriterQueueSize, c.MaxMsgLen)
	if !permit {
		return nil, errors.New(msg)
	}
	return node, nil
}

func (c *Client) serve(conn *nodeNet.Connect) {
	c.onConnect(conn)
	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			_ = conn.Close()
			c.onClose(conn, err)
			return
		}
		msg.Hop++
		if msg.DestId != c.id {
			c.onRouteMessage(nodeNet.NewContext(conn, msg))
			continue
		}
		switch msg.Type {
		case message.MsgType_Default:
			c.onMessage(nodeNet.NewContext(conn, msg))
		case message.MsgType_Reply, message.MsgType_ReplyErr:
			c.recvLock.Lock()
			ch, ok := c.recvChan[msg.Id]
			if ok {
				ch <- msg
				delete(c.recvChan, msg.Id)
			}
			c.recvLock.Unlock()
		default:
			c.onProtocolMessage(nodeNet.NewContext(conn, msg))
		}
	}
}

func (c *Client) Id() uint32 {
	return c.id
}

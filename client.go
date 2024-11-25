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
	id                       uint32
	remoteId                 uint32
	remoteKey                []byte
	authTimeout              time.Duration
	recvChan                 map[uint32]chan *message.Message
	recvLock                 sync.Mutex
	onConnectionCallback     []func(conn iface.Conn)
	onMessageCallback        []func(ctx iface.Context)
	onCustomMessageCallback  []func(ctx iface.Context)
	onNoRouteMessageCallback []func(ctx iface.Context)
	onClosedCallback         []func(conn iface.Conn, err error)
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
	c.handleOnConnections(node)
	go c.serve(node)
	return node, nil
}

func (c *Client) authenticate(conn net.Conn) (*nodeNet.Connect, error) {
	err := defaultBasicReq.Send(conn, c.id, c.remoteId, c.remoteKey, NodeType_Base)
	if err != nil {
		return nil, err
	}
	permit, msg, err := defaultBasicResp.Receive(conn, c.authTimeout)
	if err != nil {
		return nil, err
	}
	node := nodeNet.NewConn(c.id, c.remoteId, conn, c.recvChan, &c.recvLock, new(uint32), c.ReaderBufSize, c.WriterBufSize, c.WriterQueueSize, c.MaxMsgLen, uint8(NodeType_Bridge))
	if !permit {
		return nil, errors.New(msg)
	}
	return node, nil
}

func (c *Client) serve(conn *nodeNet.Connect) {
	for {
		msg, err := conn.ReadMsg()
		if err != nil {
			_ = conn.Close()
			c.handleOnClosed(conn, err)
			return
		}
		msg.Hop++
		if msg.DestId != c.id {
			c.handleOnNoRouteMessagesMessages(nodeNet.NewContext(conn, msg, true))
			continue
		}
		switch msg.Type {
		case message.MsgType_Send:
			c.handleOnMessages(nodeNet.NewContext(conn, msg, true))
		case message.MsgType_Reply, message.MsgType_ReplyErr, message.MsgType_ReplyErrConnNotExist, message.MsgType_ReplyErrLenLimit, message.MsgType_ReplyErrCheckSum:
			c.recvLock.Lock()
			ch, ok := c.recvChan[msg.Id]
			if ok {
				ch <- msg
				delete(c.recvChan, msg.Id)
			}
			c.recvLock.Unlock()
		default:
			c.handleOnCustomMessages(nodeNet.NewContext(conn, msg, true))
		}
	}
}

func (c *Client) AddOnConnection(callback func(conn iface.Conn)) {
	c.onConnectionCallback = append(c.onConnectionCallback, callback)
}

func (c *Client) AddOnMessage(callback func(conn iface.Context)) {
	c.onMessageCallback = append(c.onMessageCallback, callback)
}

func (c *Client) AddOnCustomMessage(callback func(conn iface.Context)) {
	c.onCustomMessageCallback = append(c.onCustomMessageCallback, callback)
}

func (c *Client) AddOnNoRouteMessage(callback func(conn iface.Context)) {
	c.onNoRouteMessageCallback = append(c.onNoRouteMessageCallback, callback)
}

func (c *Client) AddOnClosed(callback func(conn iface.Conn, err error)) {
	c.onClosedCallback = append(c.onClosedCallback, callback)
}

func (c *Client) handleOnConnections(conn iface.Conn) {
	for _, callback := range c.onConnectionCallback {
		callback(conn)
	}
}

func (c *Client) handleOnMessages(ctx iface.Context) {
	for _, callback := range c.onMessageCallback {
		callback(ctx)
		if !ctx.Next() {
			return
		}
	}
}

func (c *Client) handleOnCustomMessages(ctx iface.Context) {
	for _, callback := range c.onCustomMessageCallback {
		callback(ctx)
		if !ctx.Next() {
			return
		}
	}
}

func (c *Client) handleOnNoRouteMessagesMessages(ctx iface.Context) {
	for _, callback := range c.onNoRouteMessageCallback {
		callback(ctx)
		if !ctx.Next() {
			return
		}
	}
}

func (c *Client) handleOnClosed(conn iface.Conn, err error) {
	for _, callback := range c.onClosedCallback {
		callback(conn, err)
	}
}

func (c *Client) Id() uint32 {
	return c.id
}

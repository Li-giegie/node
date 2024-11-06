package node

import (
	"errors"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	nodeNet "github.com/Li-giegie/node/net"
	"io"
	"net"
	"sync"
)

type CliConf struct {
	ReaderBufSize   int
	WriterBufSize   int
	WriterQueueSize int
	MaxMsgLen       uint32
	*ClientIdentity
}

type Client struct {
	*CliConf
	conn     net.Conn
	recvChan map[uint32]chan *message.Message
	recvLock sync.Mutex
	iface.Conn
	OnConnections     []func(conn iface.Conn)
	OnMessages        []func(ctx iface.Context)
	OnCustomMessages  []func(ctx iface.Context)
	OnNoRouteMessages []func(ctx iface.Context)
	OnCloseds         []func(conn iface.Conn, err error)
}

func NewClient(conn net.Conn, conf CliConf) iface.Client {
	c := new(Client)
	c.recvChan = make(map[uint32]chan *message.Message)
	c.conn = conn
	c.CliConf = &conf
	return c
}

func (c *Client) authenticate() (*nodeNet.Connect, error) {
	err := defaultBasicReq.Send(c.conn, c.ClientIdentity.Id, c.ClientIdentity.RemoteAuthKey, NodeType_Base)
	if err != nil {
		_ = c.conn
		return &nodeNet.Connect{}, err
	}
	rid, permit, msg, err := defaultBasicResp.Receive(c.conn, c.ClientIdentity.Timeout)
	if err != nil {
		_ = c.conn.Close()
		return &nodeNet.Connect{}, err
	}
	conn := nodeNet.NewConn(c.ClientIdentity.Id, rid, c.conn, c.recvChan, &c.recvLock, new(uint32), c.ReaderBufSize, c.WriterBufSize, c.WriterQueueSize, c.MaxMsgLen, uint8(NodeType_Bridge))
	if !permit {
		_ = c.conn.Close()
		return conn, errors.New(msg)
	}
	return conn, nil
}

func (c *Client) Start() error {
	conn, err := c.authenticate()
	if err != nil {
		c.handleOnClosed(conn, err)
		return err
	}
	c.Conn = conn
	c.handleOnConnections(conn)
	go func() {
		for {
			msg, err := conn.ReadMsg()
			if err != nil {
				if conn.IsClosed() || errors.Is(err, io.EOF) {
					err = nil
				}
				_ = conn.Close()
				c.handleOnClosed(conn, err)
				return
			}
			if msg.DestId != c.ClientIdentity.Id {
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
	}()
	return nil
}

func (c *Client) AddOnConnection(callback func(conn iface.Conn)) {
	c.OnConnections = append(c.OnConnections, callback)
}

func (c *Client) AddOnMessage(callback func(conn iface.Context)) {
	c.OnMessages = append(c.OnMessages, callback)
}

func (c *Client) AddOnCustomMessage(callback func(conn iface.Context)) {
	c.OnCustomMessages = append(c.OnCustomMessages, callback)
}

func (c *Client) AddOnNoRouteMessage(callback func(conn iface.Context)) {
	c.OnNoRouteMessages = append(c.OnNoRouteMessages, callback)
}

func (c *Client) AddOnClosed(callback func(conn iface.Conn, err error)) {
	c.OnCloseds = append(c.OnCloseds, callback)
}

func (c *Client) handleOnConnections(conn iface.Conn) {
	for _, callback := range c.OnConnections {
		callback(conn)
	}
}

func (c *Client) handleOnMessages(ctx iface.Context) {
	for _, callback := range c.OnMessages {
		callback(ctx)
		if !ctx.Next() {
			return
		}
	}
}

func (c *Client) handleOnCustomMessages(ctx iface.Context) {
	for _, callback := range c.OnCustomMessages {
		callback(ctx)
		if !ctx.Next() {
			return
		}
	}
}

func (c *Client) handleOnNoRouteMessagesMessages(ctx iface.Context) {
	for _, callback := range c.OnNoRouteMessages {
		callback(ctx)
		if !ctx.Next() {
			return
		}
	}
}

func (c *Client) handleOnClosed(conn iface.Conn, err error) {
	for _, callback := range c.OnCloseds {
		callback(conn, err)
	}
}

func (c *Client) Id() uint32 {
	return c.ClientIdentity.Id
}

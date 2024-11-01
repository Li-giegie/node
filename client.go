package node

import (
	"errors"
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
	OnConnection    func(conn Conn)            `yaml:"-" json:"-"`
	OnMessage       func(ctx Context)          `yaml:"-" json:"-"`
	OnCustomMessage func(ctx CustomContext)    `yaml:"-" json:"-"`
	OnDropMessage   func(ctx Context)          `yaml:"-" json:"-"`
	OnClose         func(id uint32, err error) `yaml:"-" json:"-"`
}

type Client struct {
	*CliConf
	conn     net.Conn
	recvChan map[uint32]chan *nodeNet.Message
	recvLock sync.Mutex
	Conn
}

func NewClient(conn net.Conn, conf *CliConf) *Client {
	c := new(Client)
	c.recvChan = make(map[uint32]chan *nodeNet.Message)
	c.conn = conn
	c.CliConf = conf
	return c
}

func (c *Client) authenticate() (*nodeNet.Connect, error) {
	err := defaultBasicReq.Send(c.conn, c.ClientIdentity.Id, c.ClientIdentity.RemoteAuthKey)
	if err != nil {
		_ = c.conn
		return nil, err
	}
	rid, permit, msg, err := defaultBasicResp.Receive(c.conn, c.ClientIdentity.Timeout)
	if err != nil {
		_ = c.conn.Close()
		return nil, err
	}
	if !permit {
		_ = c.conn.Close()
		return nil, errors.New(msg)
	}
	conn := nodeNet.NewConn(c.ClientIdentity.Id, rid, c.conn, c.recvChan, &c.recvLock, new(uint32), c.ReaderBufSize, c.WriterBufSize, c.WriterQueueSize, c.MaxMsgLen)
	return conn, nil
}

func (c *Client) Start() error {
	conn, err := c.authenticate()
	if err != nil {
		if c.OnClose != nil {
			c.OnClose(0, err)
		}
		return err
	}
	c.Conn = conn
	if c.OnConnection != nil {
		c.OnConnection(conn)
	}
	go func() {
		hBuf := make([]byte, nodeNet.MsgHeaderLen)
		for {
			msg, err := conn.ReadMsg(hBuf)
			if err != nil {
				if conn.IsClosed || errors.Is(err, io.EOF) {
					err = nil
				}
				_ = conn.Close()
				if c.OnClose != nil {
					c.OnClose(conn.RemoteId(), err)
				}
				return
			}
			if msg.DestId != c.Id {
				if c.OnDropMessage != nil {
					c.OnDropMessage(&connContext{Message: msg, Connect: conn})
				}
				continue
			}
			switch msg.Type {
			case nodeNet.MsgType_Send:
				c.OnMessage(&connContext{Message: msg, Connect: conn})
			case nodeNet.MsgType_Reply, nodeNet.MsgType_ReplyErr, nodeNet.MsgType_ReplyErrConnNotExist, nodeNet.MsgType_ReplyErrLenLimit, nodeNet.MsgType_ReplyErrCheckSum:
				c.recvLock.Lock()
				ch, ok := c.recvChan[msg.Id]
				if ok {
					ch <- msg
					delete(c.recvChan, msg.Id)
				}
				c.recvLock.Unlock()
			default:
				c.OnCustomMessage(&connContext{Message: msg, Connect: conn})
			}
		}
	}()
	return nil
}

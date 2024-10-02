package node

import (
	"errors"
	"github.com/Li-giegie/node/common"
	"net"
	"sync"
)

type Client struct {
	*Identity
	conn          net.Conn
	ReaderBufSize int
	WriterBufSize int
}

func NewClient(conn net.Conn, id *Identity) *Client {
	c := new(Client)
	c.conn = conn
	c.Identity = id
	c.ReaderBufSize = 4096
	c.WriterBufSize = 4096
	return c
}

func (c *Client) InitConn(h Handler) (Conn, error) {
	err := defaultBasicReq.Send(c.conn, c.Identity.Id, c.Identity.AccessKey)
	if err != nil {
		_ = c.conn
		return nil, err
	}
	rid, permit, msg, err := defaultBasicResp.Receive(c.conn, c.Identity.AccessTimeout)
	if err != nil {
		_ = c.conn.Close()
		return nil, err
	}
	if !permit {
		_ = c.conn.Close()
		return nil, errors.New(msg)
	}
	conn := common.NewConn(c.Identity.Id, rid, c.conn, make(map[uint32]chan *common.Message), &sync.Mutex{}, nil, nil, h, new(uint32), c.ReaderBufSize, c.WriterBufSize, 0xffffff)
	go conn.Serve()
	h.Connection(conn)
	return conn, nil
}

// DialTCP 发起tcp连接并启动服务
func DialTCP(addr string, auth *Identity, h Handler) (Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewClient(conn, auth).InitConn(h)
}

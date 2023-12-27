package node

import (
	"errors"
	"net"
	"time"
)

type iStorageMsgChan interface {
	storageMsgChan(id uint32, mshChan chan *message)
}

type connect struct {
	conn       *net.TCPConn
	Status     bool
	Id         uint64
	activation int64 //Unix second
	iStorageMsgChan
}

func newConnect(id uint64, conn *net.TCPConn, s iStorageMsgChan) *connect {
	return &connect{
		Id:              id,
		Status:          true,
		activation:      time.Now().Unix(),
		conn:            conn,
		iStorageMsgChan: s,
	}
}

func (c *connect) send(srcId uint64, typ uint8, api uint32, data []byte) error {
	m := newMsg()
	m.api = api
	m.data = data
	m.srcId = srcId
	m.typ = typ
	return c.writeMsg(m)
}

func (c *connect) request(timeout time.Duration, typ uint8, srcId, dstId uint64, api uint32, data []byte) (respData []byte, err error) {
	m := newMsg()
	m.api = api
	m.typ = typ
	m.data = data
	m.srcId = srcId
	m.dstId = dstId

	mChan := make(chan *message)
	c.storageMsgChan(m.id, mChan)
	if err = c.writeMsg(m); err != nil {
		return nil, err
	}
	select {
	case reply := <-mChan:
		defer reply.recycle()
		return reply.data, nil
	case <-time.After(timeout):
		return nil, errors.New("timeout")
	}
}

func (c *connect) writeMsg(m *message) error {
	c.activation = time.Now().Unix()
	return writeMessage(c.conn, m)
}

func (c *connect) reply(m *message, typ uint8, data []byte) error {
	m.typ = typ
	m.data = data
	m.srcId, m.dstId = m.dstId, m.srcId
	return writeMessage(c.conn, m)
}

func (c *connect) close(nowait ...bool) {
	if c.conn != nil {
		if len(nowait) > 0 && nowait[0] {
			_ = c.conn.SetLinger(0)
		}
		_ = c.conn.Close()
	}
}

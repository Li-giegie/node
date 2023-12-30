package node

import (
	"errors"
	"log"
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

func (c *connect) send(srcId, dstId uint64, typ uint8, api uint32, data []byte) error {
	return c.writeMsg(newMsg(srcId, dstId, typ, api, data))
}

func (c *connect) request(timeout time.Duration, srcId, dstId uint64, typ uint8, api uint32, data []byte) (respData []byte, err error) {
	m := newMsg(srcId, dstId, typ, api, data)
	mChan := make(chan *message)
	c.storageMsgChan(m.id, mChan)
	if err = c.writeMsg(m); err != nil {
		return nil, err
	}
	select {
	case reply := <-mChan:
		log.Println("debug ", reply.String())
		defer reply.recycle()
		if reply.typ == msgType_ReplyErr {
			return decodeErrReplyMsgData(reply.data)
		}
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

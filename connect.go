package node

import (
	"errors"
	"net"
	"time"
)

type connect struct {
	conn       *net.TCPConn
	Status     bool
	Id         uint64
	activation int64
	iMessageChan
}

func newConnect(id uint64, conn *net.TCPConn, mc iMessageChan) *connect {
	return &connect{
		Id:           id,
		Status:       true,
		conn:         conn,
		activation:   time.Now().Unix(),
		iMessageChan: mc,
	}
}

func (c *connect) send(srcId, dstId uint64, typ uint8, api uint32, data []byte) error {
	return c.writeMsg(newMsg(srcId, dstId, typ, api, data))
}

func (c *connect) request(timeout time.Duration, srcId, dstId uint64, typ uint8, api uint32, data []byte) (msg *message, err error) {
	m := newMsg(srcId, dstId, typ, api, data)
	mChan := make(chan *message)
	c.AddMsgChan(m.id, mChan)
	defer func() {
		c.DeleteMsgChan(m.id)
		close(mChan)
		mChan = nil
	}()
	if err = c.writeMsg(m); err != nil {
		return nil, err
	}
	select {
	case reply := <-mChan:
		if reply.typ == msgType_ReplyErr {
			data, err = decodeErrReplyMsgData(reply.data)
			reply.data = data
			return reply, err
		}
		return reply, nil
	case <-time.After(timeout):
		return nil, errors.New("timeout")
	}
}

func (c *connect) writeMsg(m *message) error {
	_, err := c.conn.Write(m.marshal())
	m.recycle()
	c.activation = time.Now().Unix()
	return err
}

func (c *connect) close(nowait ...bool) {
	if c.conn != nil {
		if len(nowait) > 0 && nowait[0] {
			_ = c.conn.SetLinger(0)
		}
		_ = c.conn.Close()
	}
}

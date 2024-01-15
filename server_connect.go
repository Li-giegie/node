package node

import (
	"errors"
	utils "github.com/Li-giegie/go-utils"
	"log"
	"net"
	"time"
)

type ISrvConn interface {
	Start()
	Close(nowait ...bool)
	Send(api uint32, data []byte) error
	Request(timeout time.Duration, api uint32, data []byte) (replyData []byte, err error)
	Forward(timeout time.Duration, dstId uint64, api uint32, data []byte) (replyData []byte, err error)
	GetId() uint64
}

type connectEventType uint8

const (
	connectEventType_Close connectEventType = 1 + iota
	connectEventType_TimeOutClose
	connectEventType_processClose
)

var connectEventMap = map[connectEventType]string{
	connectEventType_Close:        "connect close event",
	connectEventType_TimeOutClose: "connect timeout close event",
	connectEventType_processClose: "connect process close event",
}

type iConnMgmt interface {
	GetConnect(id uint64) (ISrvConn, bool)
	ConnectEvent(cet connectEventType, arg ...interface{})
	process(ctx *srvConnCtx) error
	ServerId() uint64
}

type srvConn struct {
	msgChan *utils.MapUint32
	apis    []uint32
	iConnMgmt
	*connect
}

func newSrvConn(id uint64, tConn *net.TCPConn, cm iConnMgmt) *srvConn {
	conn := new(srvConn)
	conn.msgChan = utils.NewMapUint32()
	conn.iConnMgmt = cm
	conn.connect = newConnect(id, tConn, conn)
	return conn
}

type srvConnCtx struct {
	msg  *message
	conn *srvConn
}

func (c *srvConn) GetId() uint64 {
	return c.Id
}

func (c *srvConn) Start() {
	var _err error
	defer c.ConnectEvent(connectEventType_processClose, _err)
	for c.Status {
		msg, err := c.read()
		if err != nil {
			c.Status = false
			_err = err
			return
		}
		c.activation = time.Now().Unix()
		err = c.process(&srvConnCtx{
			msg:  msg,
			conn: c,
		})
		if err != nil {
			_err = err
			return
		}
	}
	return
}

func (c *srvConn) read() (*message, error) {
	buf, err := readAtLeast(c.conn, msg_headerLen)
	if err != nil {
		return nil, err
	}
	m := msgPool.Get().(*message)
	dl := m.header.unmarshal(buf)
	if m.srcId != c.connect.Id {
		log.Println("invalid msg drop: ", m.String())
		return nil, errors.New("invalid message drop: " + m.String())
	}
	m.data, err = readAtLeast(c.conn, int(dl))
	return m, err
}

func (c *srvConn) Send(api uint32, data []byte) error {
	return c.send(c.ServerId(), c.Id, msgType_Send, api, data)
}

func (c *srvConn) Request(timeout time.Duration, api uint32, data []byte) (replyData []byte, err error) {
	msg, err := c.request(timeout, c.ServerId(), c.Id, msgType_Send, api, data)
	defer msg.recycle()
	return msg.data, err
}

func (c *srvConn) Forward(timeout time.Duration, dstId uint64, api uint32, data []byte) (replyData []byte, err error) {
	conn, ok := c.GetConnect(dstId)
	if !ok {
		return nil, ErrConnNotExist
	}
	return conn.Request(timeout, api, data)
}

func (c *srvConn) storageMsgChan(id uint32, mshChan chan *message) {
	c.msgChan.Set(id, mshChan)
}

func (c *srvConn) delMsgChan(id uint32) {
	c.msgChan.Delete(id)
}

func (c *srvConn) Close(nowait ...bool) {
	if len(nowait) > 0 && nowait[0] {
		c.ConnectEvent(connectEventType_Close, c, true)
		return
	}
	c.ConnectEvent(connectEventType_Close, c, false)
}

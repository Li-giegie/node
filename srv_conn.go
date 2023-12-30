package node

import (
	utils "github.com/Li-giegie/go-utils"
	"log"
	"net"
	"time"
)

type ISrvConn interface {
	Start() error
	Close(nowait ...bool)
	Send(api uint32, data []byte) error
	Request(timeout time.Duration, api uint32, data []byte) (replyData []byte, err error)
	reply(m *message, typ uint8, data []byte) error
	Forward(timeout time.Duration, dstId uint64, api uint32, data []byte) (replyData []byte, err error)
	GetId() uint64
}

type connectEventType uint8

const (
	connectEventType_Close connectEventType = 1 + iota
	connectEventType_TimeOutClose
)

var connectEventMap = map[connectEventType]string{
	connectEventType_Close:        "connect close event",
	connectEventType_TimeOutClose: "connect timeout close event",
}

type iConnMgmt interface {
	GetConnect(id uint64) (*srvConn, bool)
	ConnectEvent(cet connectEventType, arg ...interface{})
	Invoke(args interface{}) error
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

func (c *srvConn) Start() error {
	defer c.Close()
	for c.Status {
		msg, err := readMessage(c.conn)
		if err != nil {
			c.Status = false
			return err
		}
		c.activation = time.Now().Unix()
		if err = c.Invoke(&srvConnCtx{
			msg:  msg,
			conn: c,
		}); err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

func (c *srvConn) Send(api uint32, data []byte) error {
	return c.send(c.ServerId(), c.Id, msgType_Send, api, data)
}

func (c *srvConn) Request(timeout time.Duration, api uint32, data []byte) (replyData []byte, err error) {
	return c.request(timeout, c.ServerId(), c.Id, msgType_Send, api, data)
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

func (c *srvConn) Close(nowait ...bool) {
	if len(nowait) > 0 && nowait[0] {
		c.ConnectEvent(connectEventType_Close, c, true)
		return
	}
	c.ConnectEvent(connectEventType_Close, c, false)
}

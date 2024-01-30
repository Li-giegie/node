package node

import (
	"errors"
	"log"
	"net"
	"time"
)

type ISrvConn interface {
	start()
	Close(nowait ...bool)                                                                 //断开连接，可选值nowait表示是否立即关闭，不等待数据是否发送接收完毕
	Send(api uint32, data []byte) error                                                   //用于发送数据到当前连接中，不需要回复
	Request(timeout time.Duration, api uint32, data []byte) (replyData []byte, err error) //用于发送数据到当前连接中，等待回复
	Forward(ctx *Context, api uint32, data []byte) (err error)                            //将Context内容转发到当前的连接中，当前连接如果具有回复内容将直接回复到发起请求的连接,并提供重写api、data能力，返回值error连接发送成功是否，如果有错误即连接断开，一旦发送成功响应不会返回到代理端
	GetId() uint64                                                                        //获取连接的Id
	GetStatus() bool                                                                      //获取连接状态，true表示可用，false表示不可用
}

type connectEventType uint8

const (
	connectEventType_Close connectEventType = 1 + iota
	connectEventType_TimeOutClose
	connecteventtypeProcessclose
)

var connectEventMap = map[connectEventType]string{
	connectEventType_Close:        "connect close event",
	connectEventType_TimeOutClose: "connect timeout close event",
	connecteventtypeProcessclose:  "connect process close event",
}

type iServer interface {
	ConnectEvent(cet connectEventType, arg ...interface{})
	process(ctx *srvConnCtx) error
	ServerId() uint64
	getMaxConnectionIdle() time.Duration
}

type srvConn struct {
	apis []uint32
	iMessageChan
	iServer
	*connect
}

func newSrvConn(id uint64, tConn *net.TCPConn, cm iServer) *srvConn {
	conn := new(srvConn)
	conn.iMessageChan = newMessageChan()
	conn.iServer = cm
	conn.connect = newConnect(id, tConn, conn)
	return conn
}

type srvConnCtx struct {
	msg  *message
	conn *srvConn
}

func newSrvConnCtx(m *message, conn *srvConn) *srvConnCtx {
	return &srvConnCtx{
		msg:  m,
		conn: conn,
	}
}

func (c *srvConn) GetId() uint64 {
	return c.Id
}

func (c *srvConn) GetStatus() bool {
	return c.Status
}

func (c *srvConn) start() {
	var _err error
	defer c.ConnectEvent(connecteventtypeProcessclose, c, false, _err)
	for c.Status {
		if _err = c.conn.SetReadDeadline(time.Now().Add(c.getMaxConnectionIdle())); _err != nil {
			log.Println("deadline err: ", _err)
			return
		}
		msg, err := c.read()
		if err != nil {
			c.Status = false
			_err = err
			return
		}
		if _err = c.process(newSrvConnCtx(msg, c)); _err != nil {
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
		return nil, errors.New("invalid message drop: " + m.String())
	}
	m.data, err = readAtLeast(c.conn, int(dl))
	return m, err
}

func (c *srvConn) Send(api uint32, data []byte) error {
	_ = c.conn.SetDeadline(time.Now().Add(c.getMaxConnectionIdle()))
	return c.send(c.ServerId(), c.Id, msgType_Send, api, data)
}

func (c *srvConn) Request(timeout time.Duration, api uint32, data []byte) (replyData []byte, err error) {
	msg, err := c.request(timeout, c.ServerId(), c.Id, msgType_Send, api, data)
	defer msg.recycle()
	_ = c.conn.SetDeadline(time.Now().Add(c.getMaxConnectionIdle()))
	return msg.data, err
}

func (c *srvConn) Forward(ctx *Context, api uint32, data []byte) (err error) {
	ctx.dstId = c.Id
	ctx.api = api
	ctx.data = data
	_ = c.conn.SetDeadline(time.Now().Add(c.getMaxConnectionIdle()))
	return c.writeMsg(ctx.message)
}

func (c *srvConn) Close(nowait ...bool) {
	if len(nowait) > 0 && nowait[0] {
		c.ConnectEvent(connectEventType_Close, c, true)
		return
	}
	c.ConnectEvent(connectEventType_Close, c, false)
}

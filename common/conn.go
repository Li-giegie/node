package common

import (
	"bufio"
	ctx "context"
	"encoding/binary"
	"errors"
	"net"
	"sync/atomic"
	"time"
)

const (
	ConnStateTypeOnClose uint8 = iota
	ConnStateTypeOnConnect
	ConnStateTypeOnError
)

type Conn interface {
	//Request 发起一个请求，得到一个响应
	Request(ctx ctx.Context, data []byte) ([]byte, error)
	//Forward 转发一个请求到目的连接中，得到一个响应
	Forward(ctx ctx.Context, destId uint16, data []byte) ([]byte, error)
	Write(data []byte) (n int, err error)
	WriteTo(dst uint16, data []byte) (n int, err error)
	Close() error
	//State ConnStateTypeOnClose=0、ConnStateTypeOnConnect=1、ConnStateTypeOnError
	State() uint8
	//WriteMsg 适用于自定义消息类型，当消息Type不是内部定义的类型时消息的响应在CustomHandle回调中触发。标准消息类型应该使用Send、Request、Forward方法，
	WriteMsg(m *Message) (err error)
	LocalId() uint16
	RemoteId() uint16
	//Activate unix mill
	Activate() int64
}

type Connections interface {
	GetConn(id uint16) (Conn, bool)
	GetConns() []Conn
}

type Handler interface {
	// Handle 接收到标准类型消息时触发回调
	Handle(ctx Context)
	// ErrHandle 发送失败触发的回调
	ErrHandle(ctx ErrContext, err error)
	// CustomHandle 接收到自定义类型消息时触发回调
	CustomHandle(ctx CustomContext)
	// Disconnect 连接断开触发回调
	Disconnect(id uint16, err error)
}

func NewConn(localId, remoteId uint16, conn net.Conn, r *MsgReceiver, conns Connections, route Router, h Handler) (c *Connect) {
	c = new(Connect)
	c.remoteId = remoteId
	c.MsgReceiver = r
	c.localId = localId
	c.conn = conn
	c.activate = time.Now().UnixMilli()
	c.MaxMsgLen = 0x00FFFFFF
	c.ReadBuffSize = 4096
	c.Router = route
	c.Connections = conns
	c.Handler = h
	return c
}

type Connect struct {
	state        uint8
	localId      uint16
	remoteId     uint16
	activate     int64
	err          error
	MaxMsgLen    uint32
	ReadBuffSize int
	conn         net.Conn
	msgCounter   uint32
	*MsgReceiver
	Router
	Connections
	Handler
}

func (c *Connect) Activate() int64 {
	return c.activate
}

func copyMsg(m *Message) *Message {
	msg := new(Message)
	msg.Id = m.Id
	msg.Type = m.Type
	msg.SrcId = m.SrcId
	msg.DestId = m.DestId
	msg.Data = make([]byte, len(m.Data))
	copy(msg.Data, m.Data)
	return msg
}

// Serve 开启服务
func (c *Connect) Serve() {
	var err error
	defer func() {
		_ = c.Close()
		c.Disconnect(c.remoteId, err)
	}()
	c.state = ConnStateTypeOnConnect
	headerBuf := make([]byte, MsgHeaderLen)
	reader := bufio.NewReaderSize(c.conn, c.ReadBuffSize)
	for {
		msg := new(Message)
		err = msg.Decode(reader, headerBuf, c.MaxMsgLen)
		if err != nil {
			err = c.connectionErr(ctx.WithValue(ctx.Background(), "msg", msg), err)
			return
		}
		c.activate = time.Now().UnixMilli()
		// 非本地节点
		if msg.DestId != c.localId {
			// 优先转发到本地连接
			if c.Connections != nil {
				if conn, exist := c.Connections.GetConn(msg.DestId); exist {
					if err = conn.WriteMsg(msg); err == nil {
						continue
					}
				}
			}
			// 本地连接不存在，转发对用路由
			if c.Router != nil {
				// 获取能到达目的路由的全部节点
				nextList := c.Router.GetDstRoutes(msg.DestId)
				success := false
				for i := 0; i < len(nextList); i++ {
					conn, exist := c.Connections.GetConn(nextList[i].Next)
					if !exist {
						c.Router.DeleteRoute(msg.DestId, nextList[i].Next, nextList[i].Hop, nextList[i].ParentNode)
						continue
					}
					if err = conn.WriteMsg(msg); err != nil {
						c.Router.DeleteRoute(msg.DestId, nextList[i].Next, nextList[i].Hop, nextList[i].ParentNode)
						continue
					}
					success = true
					break
				}
				if success {
					continue
				}
				if len(nextList) > 0 {
					c.Router.DeleteRouteAll(msg.DestId)
				}
			}
			// 本地节点、路由均为目的节点，返回错误
			if err = c.WriteMsg(msg.ErrReply(MsgType_ReplyErrConnNotExist, c.localId)); err != nil {
				c.ErrHandle(&context{Message: msg, Connect: c}, &ErrWrite{err: err})
			}
			continue
		}
		switch msg.Type {
		case MsgType_Send:
			c.Handle(&context{Message: msg, Connect: c})
		case MsgType_Reply, MsgType_ReplyErr, MsgType_ReplyErrConnNotExist, MsgType_ReplyErrLenLimit, MsgType_ReplyErrCheckSum:
			if !c.SetMsgChan(msg) {
				c.ErrHandle(&context{Message: msg, Connect: c}, DEFAULT_ErrDrop)
			}
		default:
			c.CustomHandle(&context{Message: msg, Connect: c})
		}
	}
}

func (c *Connect) LocalId() uint16 {
	return c.localId
}
func (c *Connect) RemoteId() uint16 {
	return c.remoteId
}

func (c *Connect) State() uint8 {
	return c.state
}

func (c *Connect) Request(ctx ctx.Context, data []byte) ([]byte, error) {
	req := new(Message)
	req.Id = atomic.AddUint32(&c.msgCounter, 1)
	req.SrcId = c.localId
	req.DestId = c.remoteId
	req.Type = MsgType_Send
	req.Data = data
	return c.request(ctx, req)
}

// Forward only client use
func (c *Connect) Forward(ctx ctx.Context, destId uint16, data []byte) ([]byte, error) {
	req := new(Message)
	req.Id = atomic.AddUint32(&c.msgCounter, 1)
	req.SrcId = c.localId
	req.DestId = destId
	req.Type = MsgType_Send
	req.Data = data
	return c.request(ctx, req)
}

func (c *Connect) Write(data []byte) (n int, err error) {
	return c.WriteTo(c.remoteId, data)
}

func (c *Connect) WriteTo(dst uint16, data []byte) (n int, err error) {
	if dst == c.localId {
		return 0, ErrWriteYourself
	}
	msg := new(Message)
	msg.Id = atomic.AddUint32(&c.msgCounter, 1)
	msg.SrcId = c.localId
	msg.DestId = dst
	msg.Type = MsgType_Send
	msg.Data = data
	n, err = c.write(msg.Encode())
	return n, err
}

var ErrWriteYourself = errors.New("can't send it to yourself")

func (c *Connect) WriteMsg(m *Message) (err error) {
	if m.DestId == c.localId {
		return ErrWriteYourself
	}
	_, err = c.write(m.Encode())
	return
}

func (c *Connect) write(b []byte) (n int, err error) {
	if len(b)-MsgHeaderLen > int(c.MaxMsgLen) {
		return 0, DEFAULT_ErrMsgLenLimit
	}
	return c.conn.Write(b)
}

func (c *Connect) request(ctx ctx.Context, req *Message) ([]byte, error) {
	reqId := req.Id
	respChan := c.CreateMsgChan(reqId)
	err := c.WriteMsg(req)
	if err != nil {
		c.DeleteMsgChan(reqId)
		return nil, err
	}
	select {
	case <-ctx.Done():
		c.DeleteMsgChan(reqId)
		return nil, ctx.Err()
	case resp := <-respChan:
		c.DeleteMsgChan(reqId)
		data := resp.Data
		typ := resp.Type
		switch typ {
		case MsgType_ReplyErrConnNotExist:
			return nil, DEFAULT_ErrConnNotExist
		case MsgType_ReplyErrLenLimit:
			return nil, DEFAULT_ErrMsgLenLimit
		case MsgType_ReplyErrCheckSum:
			return nil, DEFAULT_ErrMsgLenLimit
		case MsgType_ReplyErr:
			n := binary.LittleEndian.Uint16(data)
			if n > limitErrLen {
				return data[2:], nil
			}
			n += 2
			return data[n:], &ErrReplyError{b: data[2:n]}
		default:
			return data, nil
		}
	}
}

func (c *Connect) connectionErr(ctx ctx.Context, err error) error {
	switch c.state {
	case ConnStateTypeOnClose:
		return nil
	case ConnStateTypeOnError:
		return c.err
	default:
		msg, ok := ctx.Value("msg").(*Message)
		typ := MsgType_ReplyErrCheckSum
		switch err.(type) {
		case *ErrMsgCheck:
			c.state = ConnStateTypeOnError
			c.err = err
		case *ErrMsgLenLimit:
			c.state = ConnStateTypeOnError
			c.err = err
			typ = MsgType_ReplyErrLenLimit
		default:
			c.state = ConnStateTypeOnError
			c.err = err
			return err
		}
		if ok {
			if err = c.WriteMsg(msg.ErrReply(typ, c.localId)); err != nil {
				c.ErrHandle(&context{Message: msg, Connect: c}, err)
			}
		}
		return err
	}
}

func (c *Connect) Close() error {
	c.state = ConnStateTypeOnClose
	return c.conn.Close()
}

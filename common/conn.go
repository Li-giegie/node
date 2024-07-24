package common

import (
	"bufio"
	ctx "context"
	"encoding/binary"
	"errors"
	"net"
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
}

// MsgController 消息的创建和接受抽象接口
type MsgController interface {
	// NewMsg 创建一个消息
	NewMsg(srcId, dstId uint16, typ uint8, data []byte) *Message
	// RecycleMsg 回收消息
	RecycleMsg(m *Message)
	// DefaultMsg 创建一个0值消息，id = 0
	DefaultMsg() *Message
	// CreateMsgChan 创建一个消息接收Channel，以准备接收
	CreateMsgChan(id uint32) chan *Message
	// SetMsgChan 在匹配消息的Id中设置Chan通知消息到达
	SetMsgChan(msg *Message) bool
	// DeleteMsgChan 删除已经处理完成的Chan，或已超时的Chan，这一步是必要的
	DeleteMsgChan(id uint32) bool
}

type Handler interface {
	// Connection 同步调用，连接第一次建立成功回调
	Connection(conn Conn)
	// Handle 接收到标准类型消息时触发回调
	Handle(ctx Context)
	// ErrHandle 发送失败触发的回调
	ErrHandle(msg *Message, err error)
	// CustomHandle 接收到自定义类型消息时触发回调
	CustomHandle(ctx Context)
	// Disconnect 连接断开触发回调
	Disconnect(id uint16, err error)
}

func NewConn(localId, remoteId uint16, conn net.Conn, msgCtrl MsgController, conns Connections, route Router) (c *Connect) {
	c = new(Connect)
	c.remoteId = remoteId
	c.MsgController = msgCtrl
	c.localId = localId
	c.conn = conn
	c.activate = time.Now().UnixMilli()
	c.MaxMsgLen = 0x00FFFFFF
	c.ReadBuffSize = 4096
	c.Router = route
	c.Connections = conns
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
	MsgController
	Router
	Connections
}

func (c *Connect) Activate() int64 {
	return c.activate
}

// Serve 开启服务
func (c *Connect) Serve(handler Handler) {
	var err error
	defer func() {
		_ = c.Close()
		handler.Disconnect(c.remoteId, err)
	}()
	c.state = ConnStateTypeOnConnect
	headerBuf := make([]byte, MsgHeaderLen)
	reader := bufio.NewReaderSize(c.conn, c.ReadBuffSize)
	handler.Connection(c)
	for {
		msg := c.DefaultMsg()
		err = msg.Decode(reader, headerBuf, c.MaxMsgLen)
		if err != nil {
			err = c.connectionErr(ctx.WithValue(ctx.Background(), "msg", msg), err, handler)
			return
		}
		c.activate = time.Now().UnixMilli()
		if msg.DestId != c.localId {
			if c.Connections == nil {
				if err = c.WriteMsg(msg.ErrReply(MsgType_ReplyErrConnNotExist, c.localId)); err != nil {
					handler.ErrHandle(msg, &ErrWrite{err: err})
				}
				continue
			}
			conn, exist := c.Connections.GetConn(msg.DestId)
			if exist {
				if err = conn.WriteMsg(msg); err == nil {
					continue
				}
			}
			if c.Router != nil {
				nextList := c.Router.GetDstRoutes(msg.DestId)
				success := false
				for i := 0; i < len(nextList); i++ {
					if conn, exist = c.Connections.GetConn(nextList[i].Next); !exist {
						c.Router.deleteRoute(msg.DestId, nextList[i])
						continue
					}
					if err = conn.WriteMsg(msg); err != nil {
						c.Router.deleteRoute(msg.DestId, nextList[i])
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
			if err = c.WriteMsg(msg.ErrReply(MsgType_ReplyErrConnNotExist, c.localId)); err != nil {
				handler.ErrHandle(msg, &ErrWrite{err: err})
			}
			continue
		}
		switch msg.Type {
		case MsgType_Send:
			handler.Handle(&context{Message: msg, writerMsg: c})
		case MsgType_Reply, MsgType_ReplyErr, MsgType_ReplyErrConnNotExist, MsgType_ReplyErrLenLimit, MsgType_ReplyErrCheckSum:
			if !c.SetMsgChan(msg) {
				handler.ErrHandle(msg, DEFAULT_ErrDrop)
			}
		case MsgType_PushErrAuthFailIdExist:
			err = DEFAULT_ErrAuthIdExist
			return
		default:
			handler.CustomHandle(&context{Message: msg, writerMsg: c})
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
	req := c.NewMsg(c.localId, c.remoteId, MsgType_Send, data)
	return c.request(ctx, req)
}

var ErrForwardYourself = errors.New("can not forward yourself")

// Forward only client use
func (c *Connect) Forward(ctx ctx.Context, destId uint16, data []byte) ([]byte, error) {
	if destId == c.localId {
		return nil, ErrForwardYourself
	}
	req := c.NewMsg(c.localId, destId, MsgType_Send, data)
	return c.request(ctx, req)
}

func (c *Connect) Write(data []byte) (n int, err error) {
	return c.WriteTo(c.remoteId, data)
}

func (c *Connect) WriteTo(dst uint16, data []byte) (n int, err error) {
	msg := c.NewMsg(c.localId, dst, MsgType_Send, data)
	n, err = c.conn.Write(msg.Encode())
	c.RecycleMsg(msg)
	return n, err
}

func (c *Connect) WriteMsg(m *Message) (err error) {
	_, err = c.conn.Write(m.Encode())
	return
}

func (c *Connect) request(ctx ctx.Context, req *Message) ([]byte, error) {
	respChan := c.CreateMsgChan(req.Id)
	err := c.WriteMsg(req)
	if err != nil {
		c.DeleteMsgChan(req.Id)
		c.RecycleMsg(req)
		return nil, err
	}
	select {
	case <-ctx.Done():
		c.DeleteMsgChan(req.Id)
		c.RecycleMsg(req)
		return nil, ctx.Err()
	case resp := <-respChan:
		data := resp.Data
		typ := resp.Type
		c.DeleteMsgChan(req.Id)
		c.RecycleMsg(req)
		c.RecycleMsg(resp)
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

func (c *Connect) connectionErr(ctx ctx.Context, err error, handler Handler) error {
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
				handler.ErrHandle(msg, err)
			}
		}
		return err
	}
}

func (c *Connect) Close() error {
	c.state = ConnStateTypeOnClose
	return c.conn.Close()
}

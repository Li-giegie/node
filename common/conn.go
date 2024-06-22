package common

import (
	"bufio"
	"context"
	"errors"
	"net"
	"time"
)

type Conn interface {
	//Request 发起一个请求，得到一个响应
	Request(ctx context.Context, data []byte) ([]byte, error)
	//Forward 转发一个请求到目的连接中，得到一个响应
	Forward(ctx context.Context, destId uint16, data []byte) ([]byte, error)
	//Send 仅发送数据
	Send(data []byte) (err error)
	Close() error
	State() ConnStateType
	//WriteMsg 应该使用Send、Request、Forward方法，如果发送成功有响应会被丢弃，无法的到响应。
	WriteMsg(m *Message) (err error)
	Id() uint16
	//Activate unix mill
	Activate() int64
}

type Connections interface {
	GetConn(id uint16) (Conn, bool)
}

type Handler interface {
	// Handle 接收到标准类型消息时触发回调
	Handle(ctx *Context)
	// ErrHandle 发送失败触发的回调
	ErrHandle(msg *Message)
	// DropHandle 接收到超时消息时触发回调
	DropHandle(msg *Message)
	// CustomHandle 接收到自定义类型消息时触发回调
	CustomHandle(ctx *Context)
	// Disconnect 连接断开触发回调
	Disconnect(id uint16, err error)
}

type ConnStateType uint8

const (
	ConnStateTypeOnClose = iota
	ConnStateTypeOnConnect
	ConnStateTypeOnError
)

func NewConn(localId, remoteId uint16, c net.Conn, co *MsgPool, mr *MsgReceiver, conns Connections, maxMsgLen uint32) *Connect {
	conn := new(Connect)
	conn.state = ConnStateTypeOnConnect
	conn.localId = localId
	conn.remoteId = remoteId
	conn.Conn = c
	conn.activate = time.Now().UnixMilli()
	conn.MsgReceiver = mr
	conn.MsgPool = co
	conn.r = bufio.NewReaderSize(c, 4096)
	conn.Connections = conns
	if conns == nil {
		conn.Connections = emptyConns{}
	}
	conn.maxMsgLen = maxMsgLen
	return conn
}

type Connect struct {
	state     ConnStateType
	localId   uint16
	remoteId  uint16
	activate  int64
	err       error
	maxMsgLen uint32
	net.Conn
	r *bufio.Reader
	*MsgReceiver
	*MsgPool
	Connections
}

func (c *Connect) Activate() int64 {
	return c.activate
}

// Serve 开启服务
func (c *Connect) Serve(h Handler) (err error) {
	defer h.Disconnect(c.remoteId, err)
	headerBuf := make([]byte, MsgHeaderLen)
	for {
		msg := c.MsgPool.Default()
		err = msg.Decode(c.r, headerBuf, c.maxMsgLen)
		if err != nil {
			return c.connectionErr(context.WithValue(context.Background(), "msg", msg), err, h)
		}
		c.activate = time.Now().UnixMilli()
		if msg.DestId != c.localId && msg.DestId != 0 {
			conn, ok := c.Connections.GetConn(msg.DestId)
			if !ok {
				if err = c.WriteMsg(msg.ErrReply(MsgType_ReplyErrConnNotExist, c.localId)); err != nil {
					h.ErrHandle(msg)
				}
			} else {
				if err = conn.WriteMsg(msg); err != nil {
					if err = c.WriteMsg(msg.ErrReply(MsgType_ReplyErrConnNotExist, c.localId)); err != nil {
						h.ErrHandle(msg)
					}
				}
			}
			continue
		}
		switch msg.Type {
		case MsgType_Send:
			h.Handle(NewContext(msg, c))
		case MsgType_Reply, MsgType_ReplyErrConnNotExist, MsgType_ReplyErrLenLimit, MsgType_ReplyErrCheckSum:
			if !c.MsgReceiver.SetMsg(msg) {
				h.DropHandle(msg)
			}
		case MsgType_PushErrAuthFail:
			return DEFAULT_ErrAuth
		default:
			h.CustomHandle(NewContext(msg, c))
		}
	}
}

func (c *Connect) Id() uint16 {
	return c.localId
}

func (c *Connect) State() ConnStateType {
	return c.state
}

func (c *Connect) Request(ctx context.Context, data []byte) ([]byte, error) {
	req := c.MsgPool.New(c.localId, c.remoteId, MsgType_Send, data)
	return c.request(ctx, req)
}

var ErrForwardYourself = errors.New("can not forward yourself")

// Forward only client use
func (c *Connect) Forward(ctx context.Context, destId uint16, data []byte) ([]byte, error) {
	if destId == c.localId {
		return nil, ErrForwardYourself
	}
	req := c.MsgPool.New(c.localId, destId, MsgType_Send, data)
	return c.request(ctx, req)
}

// Send no response data
func (c *Connect) Send(data []byte) (err error) {
	req := c.MsgPool.New(c.localId, c.remoteId, MsgType_Send, data)
	err = c.WriteMsg(req)
	c.MsgPool.Recycle(req)
	return err
}

func (c *Connect) WriteMsg(m *Message) (err error) {
	_, err = c.Conn.Write(m.Encode())
	return
}

func (c *Connect) request(ctx context.Context, req *Message) ([]byte, error) {
	respChan := c.MsgReceiver.Create(req.Id)
	err := c.WriteMsg(req)
	if err != nil {
		c.MsgReceiver.Delete(req.Id)
		c.MsgPool.Recycle(req)
		return nil, err
	}
	select {
	case <-ctx.Done():
		c.MsgReceiver.Delete(req.Id)
		c.MsgPool.Recycle(req)
		return nil, ctx.Err()
	case resp := <-respChan:
		data := resp.Data
		typ := resp.Type
		c.MsgReceiver.Delete(req.Id)
		c.MsgPool.Recycle(req)
		c.MsgPool.Recycle(resp)
		switch typ {
		case MsgType_ReplyErrConnNotExist:
			return nil, DEFAULT_ErrConnNotExist
		case MsgType_ReplyErrLenLimit:
			return nil, DEFAULT_ErrMsgLenLimit
		case MsgType_ReplyErrCheckSum:
			return nil, DEFAULT_ErrMsgLenLimit
		default:
			return data, nil
		}
	}
}

func (c *Connect) connectionErr(ctx context.Context, err error, h Handler) error {
	switch c.state {
	case ConnStateTypeOnClose:
		return nil
	case ConnStateTypeOnError:
		return c.err
	default:
		msg, ok := ctx.Value("msg").(*Message)
		var typ uint8 = MsgType_ReplyErrCheckSum
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
			h.ErrHandle(msg)
			if err = c.WriteMsg(msg.ErrReply(typ, c.localId)); err != nil {
				h.ErrHandle(msg)
			}
		}
		_ = c.Close()
		return err
	}
}

func (c *Connect) Close() error {
	c.state = ConnStateTypeOnClose
	return c.Conn.Close()
}

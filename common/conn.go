package common

import (
	"bufio"
	ctx "context"
	"errors"
	"github.com/Li-giegie/node/utils"
	"net"
	"time"
)

type Conn interface {
	//Request 发起一个请求，得到一个响应
	Request(ctx ctx.Context, data []byte) ([]byte, error)
	//Forward 转发一个请求到目的连接中，得到一个响应
	Forward(ctx ctx.Context, destId uint16, data []byte) ([]byte, error)
	//Send 仅发送数据
	Send(data []byte) (err error)
	Close() error
	State() ConnStateType
	//WriteMsg 适用于自定义消息类型，消息的响应和接收都会在CustomHandle回调中触发，标准消息类型应该使用Send、Request、Forward方法，
	//如果调用该方法发送了标准消息类型，可能造成标准请求或转发得到的响应消息是一个自定义消息回复，这显然是一个错误的回复。
	//造成这一情况的原因是框架内部维护了消息Id，调该方法给的Id是调用者自行给定的，可能与框架内部的Id起到冲突了，
	//解决方法
	//	一：调用该方法传递的类型一定不能是框架定义的标准消息类型，在github.com/Li-giegie/node/common/message.go 文件内查看标准消息类型
	//	二：获取到框架内部消息池生成的消息结构体，Conn.(*Connect).MsgPool.New()来创建消息，MsgReceiver可以用来创建、设置响应消息但是你必须在CustomHandle回调中来自行响应他。
	/*
		解决方法二示例
		发起请求示例
		//先获取到原始结构体 common.Conn.(*common.Connect)
		conn := c.Conn.(*common.Connect)
		//创建消息 srcId dstId type data
		msg := conn.MsgPool.New(conn.LocalId(), 0, 200, []byte("Custom msg"))
		//创建响应接收Chan，把消息Id告诉接收器
		replyChan := conn.MsgReceiver.Create(msg.Id)
		//发送消息
		conn.WriteMsg(msg)
		//等待接收
		replyMsg := <-replyChan

		在自定义回调中设置响应示例：
		c.Conn.(*common.Connect).MsgReceiver.SetMsg(&common.Message{
			Type:   ctx.Type(),
			Id:     ctx.Id(),
			SrcId:  ctx.SrcId(),
			DestId: ctx.DestId(),
			Data:   ctx.Data(),
		})
	*/
	WriteMsg(m *Message) (err error)
	LocalId() uint16
	RemoteId() uint16
	//Activate unix mill
	Activate() int64
}

type Connections interface {
	GetConn(id uint16) (Conn, bool)
}

type Handler interface {
	// Connection 连接被建立时触发回调
	Connection(conn net.Conn) (remoteId uint16, err error)
	// Handle 接收到标准类型消息时触发回调
	Handle(ctx Context)
	// ErrHandle 发送失败触发的回调
	ErrHandle(msg *Message)
	// DropHandle 接收到超时消息时触发回调
	DropHandle(msg *Message)
	// CustomHandle 接收到自定义类型消息时触发回调
	CustomHandle(ctx Context)
	// Disconnect 连接断开触发回调
	Disconnect(id uint16, err error)
}

type ConnStateType uint8

const (
	ConnStateTypeOnClose = iota
	ConnStateTypeOnConnect
	ConnStateTypeOnError
)

func NewConn(localId uint16, conn net.Conn, co *MsgPool, mr *MsgReceiver, conns Connections, h Handler) (c *Connect, err error) {
	c = new(Connect)
	c.remoteId, err = h.Connection(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	c.localId = localId
	c.conn = conn
	c.Handler = h
	c.activate = time.Now().UnixMilli()
	c.MsgReceiver = mr
	c.MsgPool = co
	c.MaxMsgLen = 0x00FFFFFF
	c.ReadBuffSize = 4096
	c.Connections = conns
	if c.Connections == nil {
		c.Connections = emptyConns{}
	}
	return c, nil
}

type Connect struct {
	state        ConnStateType
	localId      uint16
	remoteId     uint16
	activate     int64
	err          error
	MaxMsgLen    uint32
	ReadBuffSize int
	conn         net.Conn
	*MsgReceiver
	*MsgPool
	Connections
	Handler
}

func (c *Connect) Activate() int64 {
	return c.activate
}

// Serve 开启服务
func (c *Connect) Serve(h Handler) {
	var err error
	defer func() { h.Disconnect(c.remoteId, err) }()
	c.state = ConnStateTypeOnConnect
	headerBuf := make([]byte, MsgHeaderLen)
	reader := bufio.NewReaderSize(c.conn, c.ReadBuffSize)
	for {
		msg := c.MsgPool.Default()
		err = msg.Decode(reader, headerBuf, c.MaxMsgLen)
		if err != nil {
			err = c.connectionErr(ctx.WithValue(ctx.Background(), "msg", msg), err, h)
			return
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
			h.Handle(&context{Message: msg, WriterMsg: c})
		case MsgType_Reply, MsgType_ReplyErr, MsgType_ReplyErrConnNotExist, MsgType_ReplyErrLenLimit, MsgType_ReplyErrCheckSum:
			if !c.MsgReceiver.SetMsg(msg) {
				h.DropHandle(msg)
			}
		case MsgType_PushErrAuthFailIdExist:
			err = DEFAULT_ErrAuthIdExist
			return
		default:
			h.CustomHandle(&context{Message: msg, WriterMsg: c})
		}
	}
}

func (c *Connect) LocalId() uint16 {
	return c.localId
}
func (c *Connect) RemoteId() uint16 {
	return c.remoteId
}

func (c *Connect) State() ConnStateType {
	return c.state
}

func (c *Connect) Request(ctx ctx.Context, data []byte) ([]byte, error) {
	req := c.MsgPool.New(c.localId, c.remoteId, MsgType_Send, data)
	return c.request(ctx, req)
}

var ErrForwardYourself = errors.New("can not forward yourself")

// Forward only client use
func (c *Connect) Forward(ctx ctx.Context, destId uint16, data []byte) ([]byte, error) {
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
	_, err = c.conn.Write(m.Encode())
	return
}

func (c *Connect) request(ctx ctx.Context, req *Message) ([]byte, error) {
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
		case MsgType_ReplyErr:
			n := utils.DecodeUint24(data) + 3
			return data[n:], &ErrReplyError{b: data[3:n]}
		default:
			return data, nil
		}
	}
}

func (c *Connect) connectionErr(ctx ctx.Context, err error, h Handler) error {
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
	return c.conn.Close()
}

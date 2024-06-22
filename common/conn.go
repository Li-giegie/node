package common

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type Conn interface {
	//Serve 开启服务
	Serve() error
	//Request 发起一个请求，得到一个响应
	Request(ctx context.Context, api uint16, data []byte) ([]byte, error)
	//Forward 转发一个请求到目的连接中，得到一个响应
	Forward(ctx context.Context, destId uint16, api uint16, data []byte) ([]byte, error)
	//Send 仅发送数据
	Send(api uint16, data []byte) (err error)
	Close() error
	State() ConnStateType
	//WriteMsg 应该使用Send、Request、Forward方法，不应该调用此方法，此方法不会检查消息的有效性，如果发送成功有响应会被丢弃，无法的到响应。
	WriteMsg(m Encoder) (int, error)
	//Tick 发送一个心跳包，维持连接活跃。(interval：每隔多久检测一次连接是否超时, keepAlive：单位时间后没有收发消息，发送一次心跳包，timeoutClose：单位时间后没有收到心跳包，主动发起关闭连接，showTickPack：可选参数非空index 0为true则把心跳包打印输出到控制台，通常用于测试阶段)
	Tick(interval, keepAlive, timeoutClose time.Duration, showTickPack ...bool) error
	Id() uint16
	//Activate unix mill
	Activate() int64
}

type Handler interface {
	Handle(m *Message, conn Conn)
	GetHandleFunc(api uint16) (HandleFunc, bool)
	Submit(f func()) error
}

type ConnStateType uint8

const (
	ConnStateTypeOnClose = iota
	ConnStateTypeOnConnect
	ConnStateTypeOnError
)

func NewConn(sid, did uint16, c net.Conn, co *Constructor, mr *Receiver, h Handler, maxReceiveMsgLength uint32) Conn {
	conn := new(connect)
	conn.state = ConnStateTypeOnConnect
	conn.sId = sid
	conn.dId = did
	conn.Conn = c
	conn.activate = time.Now().UnixMilli()
	conn.Receiver = mr
	conn.Constructor = co
	conn.Handler = h
	conn.maxReceiveMsgLength = maxReceiveMsgLength
	conn.r = bufio.NewReaderSize(c, 4096)
	return conn
}

type connect struct {
	state               ConnStateType
	sId                 uint16
	dId                 uint16
	activate            int64
	showTick            bool
	err                 error
	maxReceiveMsgLength uint32
	net.Conn
	r *bufio.Reader
	Handler
	*Receiver
	*Constructor
}

func (c *connect) Activate() int64 {
	return c.activate
}

func (c *connect) Serve() error {
	hBuf := make([]byte, MESSAGE_HEADER_LEN)
	for {
		msg := c.Constructor.Default()
		err := msg.DecodeHeader(c.r, hBuf)
		if err != nil {
			if _, ok := err.(*ErrMsgCheck); ok {
				c.state = ConnStateTypeOnError
				c.err = err
				msg.Reply(MsgType_ReplyErrWithCheckInvalid, nil)
				_, _ = c.WriteMsg(msg)
				_ = c.Close()
				return err
			}
			return c.connectionErr(err)
		}
		if c.maxReceiveMsgLength > 0 && c.maxReceiveMsgLength < msg.DataLength {
			c.state = ConnStateTypeOnError
			c.err = DEFAULT_ErrMsgLenLimit
			msg.Reply(MsgType_ReplyErrWithLenLimit, nil)
			_, _ = c.WriteMsg(msg)
			_ = c.Close()
			return DEFAULT_ErrMsgLenLimit
		}
		err = msg.DecodeContent(c.r)
		if err != nil {
			return c.connectionErr(err)
		}
		c.activate = time.Now().UnixMilli()
		if msg.DestId != c.sId && msg.DestId != 0 {
			c.Handle(msg, c)
			continue
		}
		switch msg.Typ {
		case MsgType_Tick:
			hBuf[0] = MsgType_TickReply
			_, _ = c.Write(hBuf)
			if c.showTick {
				log.Println("receive tick-pack --- ")
			}
		case MsgType_TickReply:
			if c.showTick {
				log.Println("receive tick-reply --- ")
			}
		case MsgType_Send:
			h, ok := c.GetHandleFunc(msg.Api)
			if !ok {
				msg.Reply(MsgType_ReplyErrWithApiNotExist, nil)
				_, _ = c.WriteMsg(msg)
				break
			}
			err = c.Submit(func() {
				h(NewContext(msg, c))
			})
			if err != nil {
				_ = c.Close()
				return err
			}
		case MsgType_Reply, MsgType_ReplyErrWithApiNotExist, MsgType_ReplyErrWithConnectNotExist, MsgType_ReplyErrWithLenLimit, MsgType_ReplyErrWithCheckInvalid:
			if !c.Receiver.SetReceiveChan(msg) {
				log.Println("receive timeout test drop", "msg", msg.String())
			}
		}
	}
}

func (c *connect) Id() uint16 {
	return c.sId
}

func (c *connect) State() ConnStateType {
	return c.state
}

func (c *connect) Request(ctx context.Context, api uint16, data []byte) ([]byte, error) {
	req := c.Constructor.New(c.sId, c.dId, MsgType_Send, api, data)
	return c.request(ctx, req)
}

var ErrForwardYourself = errors.New("can not forward yourself")

// Forward only client use
func (c *connect) Forward(ctx context.Context, destId, api uint16, data []byte) ([]byte, error) {
	if destId == c.sId {
		return nil, ErrForwardYourself
	}
	req := c.Constructor.New(c.sId, destId, MsgType_Send, api, data)
	return c.request(ctx, req)
}

// Send no response data
func (c *connect) Send(api uint16, data []byte) (err error) {
	req := c.Constructor.New(c.sId, c.dId, MsgType_Send, api, data)
	_, err = c.WriteMsg(req)
	c.Constructor.Recycle(req)
	return err
}

func (c *connect) WriteMsg(m Encoder) (int, error) {
	return c.Conn.Write(m.Encode())
}

func (c *connect) request(ctx context.Context, req *Message) ([]byte, error) {
	respChan := c.Receiver.NewReceiveChan(req.Id)
	_, err := c.WriteMsg(req)
	if err != nil {
		c.Receiver.DelReceiveChan(req.Id)
		c.Constructor.Recycle(req)
		return nil, err
	}
	select {
	case <-ctx.Done():
		c.Receiver.DelReceiveChan(req.Id)
		c.Constructor.Recycle(req)
		return nil, ctx.Err()
	case resp := <-respChan:
		data := resp.Data
		typ := resp.Typ
		c.Receiver.DelReceiveChan(req.Id)
		c.Constructor.Recycle(req)
		c.Constructor.Recycle(resp)
		switch typ {
		case MsgType_ReplyErrWithApiNotExist:
			return nil, DEFAULT_MsgErrType_ApiNotExist_Error
		case MsgType_ReplyErrWithConnectNotExist:
			return nil, DEFAULT_MsgErrType_ConnectNotExist_Error
		case MsgType_ReplyErrWithLenLimit:
			return nil, DEFAULT_ErrMsgLenLimit
		case MsgType_ReplyErrWithCheckInvalid:
			return nil, DEFAULT_ErrMsgCheck
		default:
			return data, nil
		}
	}
}

func (c *connect) connectionErr(err error) error {
	switch c.state {
	case ConnStateTypeOnClose:
		return nil
	case ConnStateTypeOnError:
		return c.err
	default:
		c.state = ConnStateTypeOnError
		c.err = err
		return err
	}
}

func (c *connect) Close() error {
	c.state = ConnStateTypeOnClose
	return c.Conn.Close()
}

func (c *connect) Tick(interval, keepAlive, timeoutClose time.Duration, showTickPack ...bool) error {
	if len(showTickPack) > 0 && showTickPack[0] {
		c.showTick = true
	}
	return c.Submit(func() {
		now := int64(0)
		tickPack := NewTickPack().Request()
		for {
			time.Sleep(interval)
			now = time.Now().UnixMilli()
			if now >= c.activate+timeoutClose.Milliseconds() {
				_ = c.connectionErr(errors.New("timeout close"))
				_ = c.Conn.Close()
				return
			} else if now >= c.activate+keepAlive.Milliseconds() {
				_, err := c.WriteMsg(tickPack)
				if err != nil {
					_ = c.connectionErr(fmt.Errorf("send tick pack err %v", err))
					_ = c.Conn.Close()
					return
				}
				if c.showTick {
					log.Println("send tick --- ")
				}
			}
		}
	})
}

var TraceRequest = bytes.NewBuffer(make([]byte, 0, 1024000))
var TraceRequestMsg = make([]Message, 0, 100000)
var TraceLock sync.Mutex

func (c *connect) _request(ctx context.Context, req *Message) ([]byte, error) {
	respChan := c.Receiver.NewReceiveChan(req.Id)
	encode := req.Encode()
	TraceLock.Lock()
	TraceRequestMsg = append(TraceRequestMsg, *req)
	TraceRequest.Write(encode)
	_, err := c.Conn.Write(encode)
	TraceLock.Unlock()
	if err != nil {
		c.Receiver.DelReceiveChan(req.Id)
		c.Constructor.Recycle(req)
		return nil, err
	}
	select {
	case <-ctx.Done():
		c.Receiver.DelReceiveChan(req.Id)
		c.Constructor.Recycle(req)
		return nil, ctx.Err()
	case resp := <-respChan:
		data := resp.Data
		typ := resp.Typ
		c.Receiver.DelReceiveChan(req.Id)
		c.Constructor.Recycle(req)
		c.Constructor.Recycle(resp)
		switch typ {
		case MsgType_ReplyErrWithApiNotExist:
			return nil, DEFAULT_MsgErrType_ApiNotExist_Error
		case MsgType_ReplyErrWithConnectNotExist:
			return nil, DEFAULT_MsgErrType_ConnectNotExist_Error
		case MsgType_ReplyErrWithLenLimit:
			return nil, DEFAULT_ErrMsgLenLimit
		default:
			return data, nil
		}
	}
}

package common

import (
	"bufio"
	ctx "context"
	"encoding/binary"
	"errors"
	"github.com/Li-giegie/node/utils"
	"io"
	"net"
	"sync"
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
	AsyncRequest(ctx ctx.Context, data []byte, callback func(res []byte, err error))
	//Forward 转发一个请求到目的连接中，得到一个响应
	Forward(ctx ctx.Context, destId uint16, data []byte) ([]byte, error)
	AsyncForward(ctx ctx.Context, destId uint16, data []byte, callback func(res []byte, err error))
	Write(data []byte) (n int, err error)
	WriteTo(dst uint16, data []byte) (n int, err error)
	//WriteMsg 发送一条自定义类型的消息，当消息Type不是内部定义的类型时消息的响应在CustomHandle回调中触发。标准消息类型应该使用Write、Request、Forward方法，
	WriteMsg(m *Message) (n int, err error)
	Close() error
	//State ConnStateTypeOnClose=0、ConnStateTypeOnConnect=1、ConnStateTypeOnError
	State() uint8
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

func NewConn(localId, remoteId uint16, conn net.Conn, revChan map[uint32]chan *Message, lock *sync.Mutex, conns Connections, route Router, h Handler, counter *uint32, rBufSize, wBufSize, maxMsgLen int) (c *Connect) {
	c = new(Connect)
	c.remoteId = remoteId
	c.revChan = revChan
	c.lock = lock
	c.localId = localId
	c.conn = conn
	c.activate = time.Now().UnixMilli()
	c.MaxMsgLen = uint32(maxMsgLen) & 0xFFFFFF
	c.Router = route
	c.Connections = conns
	c.Handler = h
	c.counter = counter
	c.Writer = NewWriter(c.conn, wBufSize)
	c.Reader = bufio.NewReaderSize(c.conn, rBufSize)
	return c
}

type Connect struct {
	state     uint8
	localId   uint16
	remoteId  uint16
	activate  int64
	MaxMsgLen uint32
	counter   *uint32
	revChan   map[uint32]chan *Message
	lock      *sync.Mutex
	conn      net.Conn
	*Writer
	*bufio.Reader
	Router
	Connections
	Handler
}

func (c *Connect) Activate() int64 {
	return c.activate
}

func (c *Connect) Serve() {
	c.state = ConnStateTypeOnConnect
	headerBuf := make([]byte, MsgHeaderLen)
	for {
		msg, err := c.ReadMsg(headerBuf)
		if err != nil {
			c.handleServeErr(msg, err)
			return
		}
		c.activate = time.Now().UnixMilli()
		// 非本地节点
		if msg.DestId != c.localId {
			// 优先转发到本地连接
			if c.Connections != nil {
				if conn, exist := c.Connections.GetConn(msg.DestId); exist {
					if _, err = conn.WriteMsg(msg); err == nil {
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
					if _, err = conn.WriteMsg(msg); err != nil {
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
			msg.Type = MsgType_ReplyErrConnNotExist
			msg.DestId = msg.SrcId
			msg.SrcId = c.localId
			if _, err = c.WriteMsg(msg); err != nil {
				_ = c.conn.Close()
			}
			continue
		}
		switch msg.Type {
		case MsgType_Send:
			c.Handle(&context{Message: msg, Connect: c})
		case MsgType_Reply, MsgType_ReplyErr, MsgType_ReplyErrConnNotExist, MsgType_ReplyErrLenLimit, MsgType_ReplyErrCheckSum:
			c.lock.Lock()
			ch, ok := c.revChan[msg.Id]
			if ok {
				ch <- msg
				delete(c.revChan, msg.Id)
			}
			c.lock.Unlock()
			if !ok {
				c.ErrHandle(&context{Message: msg, Connect: c}, DEFAULT_ErrTimeoutMsg)
			}
		default:
			c.CustomHandle(&context{Message: msg, Connect: c})
		}
	}
}

func (c *Connect) ReadMsg(headerBuf []byte) (*Message, error) {
	_, err := io.ReadAtLeast(c.Reader, headerBuf, MsgHeaderLen)
	if err != nil {
		return nil, err
	}
	var checksum uint16
	for i := 0; i < 11; i++ {
		checksum += uint16(headerBuf[i])
	}
	if checksum != binary.LittleEndian.Uint16(headerBuf[11:]) {
		return nil, DEFAULT_ErrMsgChecksum
	}
	dataLen := utils.DecodeUint24(headerBuf[8:11])
	if dataLen > c.MaxMsgLen && c.MaxMsgLen > 0 {
		return nil, DEFAULT_ErrMsgLenLimit
	}
	var m Message
	m.Type = headerBuf[0]
	m.Id = utils.DecodeUint24(headerBuf[1:4])
	m.SrcId = binary.LittleEndian.Uint16(headerBuf[4:6])
	m.DestId = binary.LittleEndian.Uint16(headerBuf[6:8])
	if dataLen > 0 {
		m.Data = make([]byte, dataLen)
		_, err = io.ReadAtLeast(c.Reader, m.Data, int(dataLen))
	}
	return &m, err
}

func (c *Connect) WriteMsg(m *Message) (n int, err error) {
	if m.DestId == c.localId {
		return 0, DEFAULT_ErrWriteYourself
	}
	msgLen := MsgHeaderLen + len(m.Data)
	if msgLen > int(c.MaxMsgLen) && c.MaxMsgLen > 0 {
		return 0, DEFAULT_ErrMsgLenLimit
	}
	data := make([]byte, msgLen)
	data[0] = m.Type
	utils.EncodeUint24(data[1:4], m.Id)
	binary.LittleEndian.PutUint16(data[4:6], m.SrcId)
	binary.LittleEndian.PutUint16(data[6:8], m.DestId)
	utils.EncodeUint24(data[8:11], uint32(len(m.Data)))
	var checksum uint16
	for i := 0; i < 11; i++ {
		checksum += uint16(data[i])
	}
	binary.LittleEndian.PutUint16(data[11:], checksum)
	copy(data[MsgHeaderLen:], m.Data)
	return c.Writer.Write(data)
}

func (c *Connect) Request(ctx ctx.Context, data []byte) ([]byte, error) {
	return c.request(ctx, c.newMsg(data))
}

func (c *Connect) AsyncRequest(ctx ctx.Context, data []byte, callback func(res []byte, err error)) {
	go callback(c.Request(ctx, data))
}

func (c *Connect) Forward(ctx ctx.Context, destId uint16, data []byte) ([]byte, error) {
	req := c.newMsg(data)
	req.DestId = destId
	return c.request(ctx, req)
}

func (c *Connect) AsyncForward(ctx ctx.Context, destId uint16, data []byte, callback func(res []byte, err error)) {
	go callback(c.Forward(ctx, destId, data))
}

func (c *Connect) Write(data []byte) (n int, err error) {
	return c.WriteMsg(c.newMsg(data))
}

func (c *Connect) WriteTo(dst uint16, data []byte) (n int, err error) {
	m := c.newMsg(data)
	m.DestId = dst
	return c.WriteMsg(m)
}

func (c *Connect) request(ctx ctx.Context, req *Message) ([]byte, error) {
	ch := make(chan *Message, 1)
	id := req.Id
	c.lock.Lock()
	c.revChan[id] = ch
	c.lock.Unlock()
	_, err := c.WriteMsg(req)
	if err != nil {
		c.lock.Lock()
		delete(c.revChan, id)
		c.lock.Unlock()
		close(ch)
		return nil, err
	}
	select {
	case <-ctx.Done():
		c.lock.Lock()
		_ch, ok := c.revChan[id]
		if ok {
			close(_ch)
			delete(c.revChan, id)
		}
		c.lock.Unlock()
		return nil, ctx.Err()
	case resp := <-ch:
		close(ch)
		switch resp.Type {
		case MsgType_ReplyErrConnNotExist:
			return nil, DEFAULT_ErrConnNotExist
		case MsgType_ReplyErrLenLimit:
			return nil, DEFAULT_ErrMsgLenLimit
		case MsgType_ReplyErrCheckSum:
			return nil, DEFAULT_ErrMsgChecksum
		case MsgType_ReplyErr:
			n := binary.LittleEndian.Uint16(resp.Data)
			if n > maxErrReplySize {
				return resp.Data[2:], nil
			}
			n += 2
			return resp.Data[n:], &ErrReplyError{b: resp.Data[2:n]}
		default:
			return resp.Data, nil
		}
	}
}

func (c *Connect) handleServeErr(m *Message, err error) {
	defer func() {
		_ = c.conn.Close()
		c.Disconnect(c.remoteId, err)
	}()
	if c.state == ConnStateTypeOnClose || errors.Is(err, io.EOF) {
		err = nil
		return
	}
	c.state = ConnStateTypeOnError
	if errors.Is(err, DEFAULT_ErrMsgChecksum) {
		c.ErrHandle(&context{Message: m}, err)
		m.Type = MsgType_ReplyErrCheckSum
	} else if errors.Is(err, DEFAULT_ErrMsgLenLimit) {
		c.ErrHandle(&context{Message: m}, err)
		m.Type = MsgType_ReplyErrLenLimit
	} else {
		return
	}
	m.SrcId, m.DestId = c.localId, m.SrcId
	_, _ = c.WriteMsg(m)
	return
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

func (c *Connect) newMsg(data []byte) *Message {
	req := new(Message)
	req.Id = atomic.AddUint32(c.counter, 1) % 0xffffff
	req.SrcId = c.localId
	req.DestId = c.remoteId
	req.Type = MsgType_Send
	req.Data = data
	return req
}

func (c *Connect) Close() error {
	c.state = ConnStateTypeOnClose
	return c.conn.Close()
}

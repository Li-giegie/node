package common

import (
	"bufio"
	ctx "context"
	"encoding/binary"
	"errors"
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
	Forward(ctx ctx.Context, destId uint32, data []byte) ([]byte, error)
	AsyncForward(ctx ctx.Context, destId uint32, data []byte, callback func(res []byte, err error))
	Write(data []byte) (n int, err error)
	WriteTo(dst uint32, data []byte) (n int, err error)
	//WriteMsg 发送一条自定义类型的消息，当消息Type不是内部定义的类型时消息的响应在CustomHandle回调中触发。标准消息类型应该使用Write、Request、Forward方法，
	WriteMsg(m *Message) (n int, err error)
	Close() error
	//State ConnStateTypeOnClose=0、ConnStateTypeOnConnect=1、ConnStateTypeOnError
	State() uint8
	LocalId() uint32
	RemoteId() uint32
	//Activate unix mill
	Activate() int64
}

type Connections interface {
	GetConn(id uint32) (Conn, bool)
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
	Disconnect(id uint32, err error)
}

func NewConn(localId, remoteId uint32, conn net.Conn, revChan map[uint32]chan *Message, revLock *sync.Mutex, conns Connections, route Router, h Handler, msgIdCounter *uint32, rBufSize, wBufSize, writerQueueSize int, maxMsgLen uint32) (c *Connect) {
	c = new(Connect)
	c.remoteId = remoteId
	c.revChan = revChan
	c.revLock = revLock
	c.localId = localId
	c.conn = conn
	c.activate = time.Now().UnixMilli()
	c.MaxMsgLen = maxMsgLen
	c.Router = route
	c.Connections = conns
	c.Handler = h
	c.msgIdCounter = msgIdCounter
	c.WriterQueue = NewWriteQueue(conn, writerQueueSize, wBufSize)
	c.Reader = bufio.NewReaderSize(c.conn, rBufSize)
	return c
}

type Connect struct {
	state        uint8
	localId      uint32
	remoteId     uint32
	activate     int64
	MaxMsgLen    uint32
	msgIdCounter *uint32
	revChan      map[uint32]chan *Message
	revLock      *sync.Mutex
	conn         net.Conn
	*WriterQueue
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
	defer c.WriterQueue.Freed()
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
						c.Router.DeleteRoute(msg.DestId, nextList[i].Next, nextList[i].ParentNode, nextList[i].Hop)
						continue
					}
					if _, err = conn.WriteMsg(msg); err != nil {
						c.Router.DeleteRoute(msg.DestId, nextList[i].Next, nextList[i].ParentNode, nextList[i].Hop)
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
			c.revLock.Lock()
			ch, ok := c.revChan[msg.Id]
			if ok {
				ch <- msg
				delete(c.revChan, msg.Id)
			}
			c.revLock.Unlock()
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
	for i := 0; i < MsgHeaderLen-2; i++ {
		checksum += uint16(headerBuf[i])
	}
	if checksum != binary.LittleEndian.Uint16(headerBuf[MsgHeaderLen-2:]) {
		return nil, DEFAULT_ErrMsgChecksum
	}
	dataLen := binary.LittleEndian.Uint32(headerBuf[13:17])
	if dataLen > c.MaxMsgLen && c.MaxMsgLen > 0 {
		return nil, DEFAULT_ErrMsgLenLimit
	}
	var m Message
	m.Type = headerBuf[0]
	m.Id = binary.LittleEndian.Uint32(headerBuf[1:5])
	m.SrcId = binary.LittleEndian.Uint32(headerBuf[5:9])
	m.DestId = binary.LittleEndian.Uint32(headerBuf[9:13])
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
	binary.LittleEndian.PutUint32(data[1:5], m.Id)
	binary.LittleEndian.PutUint32(data[5:9], m.SrcId)
	binary.LittleEndian.PutUint32(data[9:13], m.DestId)
	binary.LittleEndian.PutUint32(data[13:17], uint32(len(m.Data)))
	var checksum uint16
	for i := 0; i < 17; i++ {
		checksum += uint16(data[i])
	}
	binary.LittleEndian.PutUint16(data[17:], checksum)
	copy(data[MsgHeaderLen:], m.Data)
	return c.WriterQueue.Write(data)
}

func (c *Connect) Request(ctx ctx.Context, data []byte) ([]byte, error) {
	return c.request(ctx, c.newMsg(data))
}

func (c *Connect) AsyncRequest(ctx ctx.Context, data []byte, callback func(res []byte, err error)) {
	go callback(c.Request(ctx, data))
}

func (c *Connect) Forward(ctx ctx.Context, destId uint32, data []byte) ([]byte, error) {
	req := c.newMsg(data)
	req.DestId = destId
	return c.request(ctx, req)
}

func (c *Connect) AsyncForward(ctx ctx.Context, destId uint32, data []byte, callback func(res []byte, err error)) {
	go callback(c.Forward(ctx, destId, data))
}

func (c *Connect) Write(data []byte) (n int, err error) {
	return c.WriteMsg(c.newMsg(data))
}

func (c *Connect) WriteTo(dst uint32, data []byte) (n int, err error) {
	m := c.newMsg(data)
	m.DestId = dst
	return c.WriteMsg(m)
}

func (c *Connect) request(ctx ctx.Context, req *Message) ([]byte, error) {
	ch := make(chan *Message, 1)
	id := req.Id
	c.revLock.Lock()
	c.revChan[id] = ch
	c.revLock.Unlock()
	_, err := c.WriteMsg(req)
	if err != nil {
		c.revLock.Lock()
		delete(c.revChan, id)
		c.revLock.Unlock()
		close(ch)
		return nil, err
	}
	select {
	case <-ctx.Done():
		c.revLock.Lock()
		_ch, ok := c.revChan[id]
		if ok {
			close(_ch)
			delete(c.revChan, id)
		}
		c.revLock.Unlock()
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

func (c *Connect) LocalId() uint32 {
	return c.localId
}

func (c *Connect) RemoteId() uint32 {
	return c.remoteId
}

func (c *Connect) State() uint8 {
	return c.state
}

func (c *Connect) newMsg(data []byte) *Message {
	req := new(Message)
	req.Id = atomic.AddUint32(c.msgIdCounter, 1)
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

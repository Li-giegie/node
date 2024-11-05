package net

import (
	"bufio"
	"context"
	"encoding/binary"
	"github.com/Li-giegie/node/message"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func NewConn(localId, remoteId uint32, conn net.Conn, revChan map[uint32]chan *message.Message, revLock *sync.Mutex, msgIdCounter *uint32, rBufSize, wBufSize, writerQueueSize int, maxMsgLen uint32, nodeType uint8) (c *Connect) {
	c = new(Connect)
	c.remoteId = remoteId
	c.revChan = revChan
	c.revLock = revLock
	c.localId = localId
	c.conn = conn
	c.activate = time.Now().UnixMilli()
	c.MaxMsgLen = maxMsgLen
	c.msgIdCounter = msgIdCounter
	c.WriterQueue = NewWriteQueue(conn, writerQueueSize, wBufSize)
	c.Reader = bufio.NewReaderSize(c.conn, rBufSize)
	c.nodeType = nodeType
	c.headerBuf = make([]byte, message.MsgHeaderLen)
	return c
}

type Connect struct {
	nodeType     uint8
	localId      uint32
	remoteId     uint32
	activate     int64
	MaxMsgLen    uint32
	msgIdCounter *uint32
	revChan      map[uint32]chan *message.Message
	revLock      *sync.Mutex
	conn         net.Conn
	isClosed     bool
	*WriterQueue
	*bufio.Reader
	headerBuf []byte
}

func (c *Connect) ReadMsg() (*message.Message, error) {
	_, err := io.ReadAtLeast(c.Reader, c.headerBuf, message.MsgHeaderLen)
	if err != nil {
		return nil, err
	}
	var checksum uint16
	for i := 0; i < message.MsgHeaderLen-2; i++ {
		checksum += uint16(c.headerBuf[i])
	}
	var m message.Message
	if checksum != binary.LittleEndian.Uint16(c.headerBuf[message.MsgHeaderLen-2:]) {
		m.SrcId, m.DestId = c.localId, m.SrcId
		m.Type = message.MsgType_ReplyErrCheckSum
		_, _ = c.WriteMsg(&m)
		return nil, DEFAULT_ErrMsgChecksum
	}
	dataLen := binary.LittleEndian.Uint32(c.headerBuf[13:17])
	if dataLen > c.MaxMsgLen && c.MaxMsgLen > 0 {
		m.SrcId, m.DestId = c.localId, m.SrcId
		m.Type = message.MsgType_ReplyErrLenLimit
		_, _ = c.WriteMsg(&m)
		return nil, DEFAULT_ErrMsgLenLimit
	}
	c.activate = time.Now().UnixMilli()

	m.Type = c.headerBuf[0]
	m.Id = binary.LittleEndian.Uint32(c.headerBuf[1:5])
	m.SrcId = binary.LittleEndian.Uint32(c.headerBuf[5:9])
	m.DestId = binary.LittleEndian.Uint32(c.headerBuf[9:13])
	if dataLen > 0 {
		m.Data = make([]byte, dataLen)
		_, err = io.ReadAtLeast(c.Reader, m.Data, int(dataLen))
	}
	return &m, err
}

func (c *Connect) WriteMsg(m *message.Message) (n int, err error) {
	if m.DestId == c.localId {
		return 0, DEFAULT_ErrWriteYourself
	}
	msgLen := message.MsgHeaderLen + len(m.Data)
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
	copy(data[message.MsgHeaderLen:], m.Data)
	return c.WriterQueue.Write(data)
}

func (c *Connect) Request(ctx context.Context, data []byte) ([]byte, error) {
	return c.request(ctx, c.InitMsg(data))
}

func (c *Connect) Forward(ctx context.Context, destId uint32, data []byte) ([]byte, error) {
	req := c.InitMsg(data)
	req.DestId = destId
	return c.request(ctx, req)
}

func (c *Connect) Write(data []byte) (n int, err error) {
	return c.WriteMsg(c.InitMsg(data))
}

func (c *Connect) WriteTo(dst uint32, data []byte) (n int, err error) {
	m := c.InitMsg(data)
	m.DestId = dst
	return c.WriteMsg(m)
}

func (c *Connect) request(ctx context.Context, req *message.Message) ([]byte, error) {
	ch := make(chan *message.Message, 1)
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
		case message.MsgType_ReplyErrConnNotExist:
			return nil, DEFAULT_ErrConnNotExist
		case message.MsgType_ReplyErrLenLimit:
			return nil, DEFAULT_ErrMsgLenLimit
		case message.MsgType_ReplyErrCheckSum:
			return nil, DEFAULT_ErrMsgChecksum
		case message.MsgType_ReplyErr:
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

func (c *Connect) Activate() int64 {
	return c.activate
}

func (c *Connect) NodeType() uint8 {
	return c.nodeType
}

func (c *Connect) LocalId() uint32 {
	return c.localId
}

func (c *Connect) RemoteId() uint32 {
	return c.remoteId
}

func (c *Connect) InitMsg(data []byte) *message.Message {
	req := new(message.Message)
	req.Id = atomic.AddUint32(c.msgIdCounter, 1)
	req.SrcId = c.localId
	req.DestId = c.remoteId
	req.Type = message.MsgType_Send
	req.Data = data
	return req
}

func (c *Connect) IsClosed() bool {
	return c.isClosed
}

func (c *Connect) Close() error {
	c.isClosed = true
	c.WriterQueue.Freed()
	return c.conn.Close()
}

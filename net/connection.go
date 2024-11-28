package net

import (
	"bufio"
	"context"
	"encoding/binary"
	"github.com/Li-giegie/node/message"
	"io"
	"math"
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
	c.activate = time.Duration(time.Now().UnixNano())
	c.maxMsgLen = maxMsgLen
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
	maxMsgLen    uint32
	msgIdCounter *uint32
	activate     time.Duration
	revChan      map[uint32]chan *message.Message
	revLock      *sync.Mutex
	conn         net.Conn
	headerBuf    []byte
	*WriterQueue
	*bufio.Reader
}

func (c *Connect) ReadMsg() (*message.Message, error) {
	_, err := io.ReadAtLeast(c.Reader, c.headerBuf, message.MsgHeaderLen)
	if err != nil {
		return nil, err
	}
	c.activate = time.Duration(time.Now().UnixNano())
	var checksum uint16
	for i := 0; i < message.MsgHeaderLen-2; i++ {
		checksum += uint16(c.headerBuf[i])
	}
	var m message.Message
	if checksum != binary.LittleEndian.Uint16(c.headerBuf[message.MsgHeaderLen-2:]) {
		m.SrcId, m.DestId = c.localId, m.SrcId
		m.Type = message.MsgType_ReplyErr
		m.Data = ErrChecksum
		_, _ = c.WriteMsg(&m)
		return nil, ErrChecksum
	}
	dataLen := binary.LittleEndian.Uint32(c.headerBuf[14:18])
	if dataLen > c.maxMsgLen && c.maxMsgLen > 0 {
		m.SrcId, m.DestId = c.localId, m.SrcId
		m.Type = message.MsgType_ReplyErr
		m.Data = ErrMaxMsgLen
		_, _ = c.WriteMsg(&m)
		return nil, ErrMaxMsgLen
	}
	m.Type = c.headerBuf[0]
	m.Hop = c.headerBuf[1]
	m.Id = binary.LittleEndian.Uint32(c.headerBuf[2:6])
	m.SrcId = binary.LittleEndian.Uint32(c.headerBuf[6:10])
	m.DestId = binary.LittleEndian.Uint32(c.headerBuf[10:14])
	if dataLen > 0 {
		m.Data = make([]byte, dataLen)
		_, err = io.ReadAtLeast(c.Reader, m.Data, int(dataLen))
	}
	return &m, err
}

func (c *Connect) WriteMsg(m *message.Message) (n int, err error) {
	if m.DestId == c.localId {
		return 0, ErrWriteMsgYourself
	}
	msgLen := message.MsgHeaderLen + len(m.Data)
	if msgLen > int(c.maxMsgLen) && c.maxMsgLen > 0 {
		return 0, ErrMaxMsgLen
	}
	data := make([]byte, msgLen)
	data[0] = m.Type
	data[1] = m.Hop
	binary.LittleEndian.PutUint32(data[2:6], m.Id)
	binary.LittleEndian.PutUint32(data[6:10], m.SrcId)
	binary.LittleEndian.PutUint32(data[10:14], m.DestId)
	binary.LittleEndian.PutUint32(data[14:18], uint32(len(m.Data)))
	var checksum uint16
	for i := 0; i < 18; i++ {
		checksum += uint16(data[i])
	}
	binary.LittleEndian.PutUint16(data[18:], checksum)
	copy(data[20:], m.Data)
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
		if resp.Type == message.MsgType_ReplyErr {
			n := binary.LittleEndian.Uint32(resp.Data[:4])
			if n == math.MaxUint32 {
				return resp.Data[4:], nil
			}
			n += 4
			return resp.Data[n:], NodeError(resp.Data[4:n])
		}
		return resp.Data, nil
	}
}

func (c *Connect) Activate() time.Duration {
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
	return &message.Message{
		Type:   message.MsgType_Send,
		Id:     atomic.AddUint32(c.msgIdCounter, 1),
		SrcId:  c.localId,
		DestId: c.remoteId,
		Data:   data,
	}
}

func (c *Connect) Close() error {
	c.WriterQueue.Freed()
	return c.conn.Close()
}

func (c *Connect) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Connect) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

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

func NewConn(localId, remoteId uint32, conn net.Conn, revChan map[uint32]chan *message.Message, revLock *sync.Mutex, msgIdSeq *uint32, rBufSize, wBufSize, writerQueueSize int, maxMsgLen uint32) *Conn {
	var c Conn
	c.remoteId = remoteId
	c.revChan = revChan
	c.revLock = revLock
	c.localId = localId
	c.conn = conn
	c.activate = time.Duration(time.Now().UnixNano())
	c.maxMsgLen = maxMsgLen
	c.msgIdSeq = msgIdSeq
	c.w = NewWriteQueue(conn, writerQueueSize, wBufSize)
	if rBufSize < 64 {
		c.r = conn
	} else {
		c.r = bufio.NewReaderSize(c.conn, rBufSize)
	}
	c.headerBuf = make([]byte, message.MsgHeaderLen)
	return &c
}

type Conn struct {
	localId   uint32
	remoteId  uint32
	maxMsgLen uint32
	msgIdSeq  *uint32
	activate  time.Duration
	revChan   map[uint32]chan *message.Message
	revLock   *sync.Mutex
	conn      net.Conn
	headerBuf []byte
	w         io.WriteCloser
	r         io.Reader
}

func (c *Conn) ReadMessage() (*message.Message, error) {
	_, err := io.ReadAtLeast(c.r, c.headerBuf, message.MsgHeaderLen)
	if err != nil {
		return nil, err
	}
	c.activate = time.Duration(time.Now().UnixNano())
	var checksum uint16
	for i := 0; i < message.MsgHeaderLen-2; i++ {
		checksum += uint16(c.headerBuf[i])
	}
	var m message.Message
	m.Type = c.headerBuf[0]
	m.Hop = c.headerBuf[1]
	m.Id = binary.LittleEndian.Uint32(c.headerBuf[2:6])
	m.SrcId = binary.LittleEndian.Uint32(c.headerBuf[6:10])
	m.DestId = binary.LittleEndian.Uint32(c.headerBuf[10:14])
	if checksum != binary.LittleEndian.Uint16(c.headerBuf[message.MsgHeaderLen-2:]) {
		m.SrcId, m.DestId = c.localId, m.SrcId
		m.Type = message.MsgType_Reply
		m.Data = []byte{byte(message.StateCode_CheckSumInvalid), byte(message.StateCode_CheckSumInvalid >> 8)}
		_ = c.SendMessage(&m)
		return nil, ErrChecksumInvalid
	}
	dataLen := binary.LittleEndian.Uint32(c.headerBuf[14:18])
	if dataLen > c.maxMsgLen && c.maxMsgLen > 0 {
		m.SrcId, m.DestId = c.localId, m.SrcId
		m.Type = message.MsgType_Reply
		m.Data = []byte{byte(message.StateCode_LengthOverflow), byte(message.StateCode_LengthOverflow >> 8)}
		_ = c.SendMessage(&m)
		return nil, ErrLengthOverflow
	}
	if dataLen > 0 {
		m.Data = make([]byte, dataLen)
		_, err = io.ReadAtLeast(c.r, m.Data, int(dataLen))
	}
	return &m, err
}

func (c *Conn) SendMessage(m *message.Message) error {
	if m.DestId == c.localId {
		return ErrWriteMsgYourself
	}
	msgLen := message.MsgHeaderLen + len(m.Data)
	if msgLen > int(c.maxMsgLen) && c.maxMsgLen > 0 {
		return ErrLengthOverflow
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
	_, err := c.conn.Write(data)
	return err
}

func (c *Conn) Send(data []byte) error {
	return c.SendMessage(&message.Message{
		Type:   message.MsgType_Default,
		Hop:    0,
		Id:     atomic.AddUint32(c.msgIdSeq, 1),
		SrcId:  c.localId,
		DestId: c.remoteId,
		Data:   data,
	})
}

func (c *Conn) SendTo(dst uint32, data []byte) error {
	return c.SendMessage(&message.Message{
		Type:   message.MsgType_Default,
		Hop:    0,
		Id:     atomic.AddUint32(c.msgIdSeq, 1),
		SrcId:  c.localId,
		DestId: dst,
		Data:   data,
	})
}

func (c *Conn) SendType(typ uint8, data []byte) error {
	return c.SendMessage(&message.Message{
		Type:   typ,
		Hop:    0,
		Id:     atomic.AddUint32(c.msgIdSeq, 1),
		SrcId:  c.localId,
		DestId: c.remoteId,
		Data:   data,
	})
}

func (c *Conn) SendTypeTo(typ uint8, dst uint32, data []byte) error {
	return c.SendMessage(&message.Message{
		Type:   typ,
		Hop:    0,
		Id:     atomic.AddUint32(c.msgIdSeq, 1),
		SrcId:  c.localId,
		DestId: dst,
		Data:   data,
	})
}

func (c *Conn) Request(ctx context.Context, data []byte) ([]byte, int16, error) {
	return c.RequestMessage(ctx, &message.Message{
		Type:   message.MsgType_Default,
		Id:     atomic.AddUint32(c.msgIdSeq, 1),
		SrcId:  c.localId,
		DestId: c.remoteId,
		Data:   data,
	})
}

func (c *Conn) RequestTo(ctx context.Context, dst uint32, data []byte) ([]byte, int16, error) {
	return c.RequestMessage(ctx, &message.Message{
		Type:   message.MsgType_Default,
		Id:     atomic.AddUint32(c.msgIdSeq, 1),
		SrcId:  c.localId,
		DestId: dst,
		Data:   data,
	})
}

func (c *Conn) RequestType(ctx context.Context, typ uint8, data []byte) ([]byte, int16, error) {
	return c.RequestMessage(ctx, &message.Message{
		Type:   typ,
		Id:     atomic.AddUint32(c.msgIdSeq, 1),
		SrcId:  c.localId,
		DestId: c.remoteId,
		Data:   data,
	})
}

func (c *Conn) RequestTypeTo(ctx context.Context, typ uint8, dst uint32, data []byte) ([]byte, int16, error) {
	return c.RequestMessage(ctx, &message.Message{
		Type:   typ,
		Id:     atomic.AddUint32(c.msgIdSeq, 1),
		SrcId:  c.localId,
		DestId: dst,
		Data:   data,
	})
}

func (c *Conn) RequestMessage(ctx context.Context, msg *message.Message) ([]byte, int16, error) {
	ch := make(chan *message.Message, 1)
	id := msg.Id
	c.revLock.Lock()
	c.revChan[id] = ch
	c.revLock.Unlock()
	err := c.SendMessage(msg)
	if err != nil {
		c.revLock.Lock()
		delete(c.revChan, id)
		c.revLock.Unlock()
		close(ch)
		return nil, 0, err
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
		return nil, message.StateCode_RequestTimeout, ctx.Err()
	case resp := <-ch:
		close(ch)
		if len(resp.Data) < 2 {
			return nil, message.StateCode_ResponseInvalid, ErrInvalidResponse
		}
		code := int16(resp.Data[0]) | int16(resp.Data[1])<<8
		switch code {
		case message.StateCode_CheckSumInvalid:
			return resp.Data[2:], code, ErrChecksumInvalid
		case message.StateCode_LengthOverflow:
			return resp.Data[2:], code, ErrLengthOverflow
		}
		return resp.Data[2:], code, nil
	}
}

func (c *Conn) Activate() time.Duration {
	return c.activate
}

func (c *Conn) LocalId() uint32 {
	return c.localId
}

func (c *Conn) RemoteId() uint32 {
	return c.remoteId
}

func (c *Conn) CreateMessage(typ uint8, src uint32, dst uint32, data []byte) *message.Message {
	return &message.Message{
		Type:   typ,
		Id:     atomic.AddUint32(c.msgIdSeq, 1),
		SrcId:  src,
		DestId: dst,
		Data:   data,
	}
}

func (c *Conn) CreateMessageId() uint32 {
	return atomic.AddUint32(c.msgIdSeq, 1)
}

func (c *Conn) Close() error {
	return c.w.Close()
}

func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

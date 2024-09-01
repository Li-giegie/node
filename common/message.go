package common

import (
	"encoding/binary"
	"fmt"
	"github.com/Li-giegie/node/utils"
	"io"
	"time"
)

// 标准消息类型
const (
	MsgType_Send uint8 = iota
	MsgType_Reply
	MsgType_ReplyErrConnNotExist
	MsgType_ReplyErrLenLimit
	MsgType_ReplyErrCheckSum
	MsgType_ReplyErr
	Null
)

const MsgHeaderLen = 1 + 3 + 2 + 2 + 3 + 2

type Message struct {
	Type   uint8
	Id     uint32
	SrcId  uint16
	DestId uint16
	Data   []byte
}

func (m *Message) String() string {
	return fmt.Sprintf("Message { type: %d, id: %v, srcId: %v, destId: %v, data: %s}", m.Type, m.Id, m.SrcId, m.DestId, m.Data)
}

func (m *Message) Encode() []byte {
	buf := make([]byte, MsgHeaderLen, MsgHeaderLen+len(m.Data))
	buf[0] = m.Type
	utils.EncodeUint24(buf[1:], m.Id)
	binary.LittleEndian.PutUint16(buf[4:], m.SrcId)
	binary.LittleEndian.PutUint16(buf[6:], m.DestId)
	dataLen := uint32(len(m.Data))
	utils.EncodeUint24(buf[8:], dataLen)
	binary.LittleEndian.PutUint16(buf[11:], uint16(m.Type)^uint16(m.Id)^m.SrcId^m.DestId^uint16(dataLen))
	return append(buf, m.Data...)
}

func (m *Message) Decode(r io.Reader, b []byte, limit uint32) (err error) {
	_, err = io.ReadAtLeast(r, b, MsgHeaderLen)
	if err != nil {
		return err
	}
	m.Type = b[0]
	m.Id = utils.DecodeUint24(b[1:])
	m.SrcId = binary.LittleEndian.Uint16(b[4:])
	m.DestId = binary.LittleEndian.Uint16(b[6:])
	dataLen := utils.DecodeUint24(b[8:])
	checkSum := binary.LittleEndian.Uint16(b[11:])
	ok := uint16(m.Type)^uint16(m.Id)^m.SrcId^m.DestId^uint16(dataLen) == checkSum
	if !ok {
		return DEFAULT_ErrMsgCheck
	}
	if limit > 0 && dataLen > limit {
		return DEFAULT_ErrMsgLenLimit
	}
	m.Data = make([]byte, dataLen)
	_, err = io.ReadFull(r, m.Data)
	return err
}

func (m *Message) DecodeWithTimeout(duration time.Duration, r io.Reader, b []byte, limit uint32) error {
	errChan := make(chan error)
	go func() {
		errChan <- m.Decode(r, b, limit)
	}()
	select {
	case err := <-errChan:
		return err
	case <-time.After(duration):
		return DEFAULT_ErrTimeout
	}
}

func (m *Message) Reply(typ uint8, data []byte) *Message {
	m.Type = typ
	m.Data = data
	m.SrcId, m.DestId = m.DestId, m.SrcId
	return m
}

// ErrReply srcId: handler Id
func (m *Message) ErrReply(typ uint8, srcId uint16) *Message {
	m.Type = typ
	m.SrcId, m.DestId = srcId, m.SrcId
	return m
}

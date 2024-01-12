package message

import (
	jeans "github.com/Li-giegie/go-jeans"
	"io"
)

const Version uint8 = 1

const (
	message_typ_auth uint8 = 1 + iota
	message_typ_auth_reply
)

type ISubMessage interface {
	init(im IMessage)
	typ() uint8
	marshal() ([]byte, error)
	unmarshal(buf []byte) error
}

type IMessage interface {
	SrcId() uint32
	DstId() uint32
}

const messageHeaderLength = 13

type message struct {
	srcId   uint32
	dstId   uint32
	typ     uint8
	dataLen uint32
	subMsg  ISubMessage
}

func newMessage(srcId, dstId uint32, sub ISubMessage) *message {
	return &message{
		srcId:  srcId,
		dstId:  dstId,
		typ:    sub.typ(),
		subMsg: sub,
	}
}

func (m *message) SrcId() uint32 {
	return m.srcId
}

func (m *message) DstId() uint32 {
	return m.dstId
}

func (m *message) marshalHeader() []byte {
	buf, err := jeans.Encode(m.srcId, m.dstId, m.typ, m.dataLen)
	if err != nil {
		panic(err)
	}
	return buf
}

func (m *message) unmarshalHeader(buf []byte) error {
	return jeans.Decode(buf, &m.srcId, &m.dstId, &m.typ, &m.dataLen)
}

func (m *message) readHeader(r io.Reader) ([]byte, error) {
	return readAtLeast(r, messageHeaderLength)
}

func (m *message) marshal() ([]byte, error) {
	buf, err := m.subMsg.marshal()
	if err != nil {
		return nil, err
	}
	m.dataLen = uint32(len(buf))
	return append(m.marshalHeader(), buf...), err
}

func (m *message) unmarshal(buf []byte) error {
	err := m.unmarshalHeader(buf)
	if err != nil {
		return err
	}
	return m.unmarshalHeader(buf[messageHeaderLength:])
}

func readAtLeast(r io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadAtLeast(r, buf, n)
	return buf, err
}

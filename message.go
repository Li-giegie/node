package node

import (
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"sync/atomic"
)

var msgCounter uint32

type Message struct {
	id       uint32
	API      uint32
	_type    uint8
	Data     []byte
	localId  string
	remoteId string
	reply    chan *Message
	buf      []byte
}

func NewMsg() *Message {
	m := new(Message)
	m.id = atomic.AddUint32(&msgCounter, 1)
	m.reply = make(chan *Message)
	return m
}

func NewMsgWithUnmarshal(b []byte) *Message {
	m := new(Message)
	m.Unmarshal(b)
	return m
}

func (m *Message) Marshal() []byte {
	buf, err := jeans.BaseTypeToBytes(m._type, m.id, m.API, m.localId, m.remoteId, m.Data)
	if err != nil {
		panic(any(err))
	}
	m.buf = buf
	return buf
}

func (m *Message) Unmarshal(buf []byte) {
	err := jeans.BytesToBaseType(buf, &m._type, &m.id, &m.API, &m.localId, &m.remoteId, &m.Data)
	if err != nil {
		panic(any(err))
	}
	m.buf = buf
}

func (m *Message) String() string {
	if m == nil {
		return "nil"
	}
	if m.isForward() {
		return fmt.Sprintf("message : id [%d] api [%d] type [%s] data %s local-id [%v] remote-id[%v]", m.id, m.API, MsgTypeMap[m._type], m.Data, m.localId, m.remoteId)
	}
	return fmt.Sprintf("message : id [%d] api [%d] type [%s] data %s", m.id, m.API, MsgTypeMap[m._type], m.Data)
}

func (m *Message) isForward() bool {
	if m._type == MsgType_ReqForward || m._type == MsgType_RespForward || m._type == MsgType_RespForwardFail || m._type == MsgType_ReqForwardFail {
		return true
	}
	return false
}

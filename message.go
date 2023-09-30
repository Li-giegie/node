package node

import (
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"sync/atomic"
)

type MessageBaseI interface {
	GetId() uint32
	GetAPI() uint32
	GetData() []byte
	GetType() uint8
	String() string
	debug()
	get() *MessageBase
}

type MessageBase struct {
	Id           uint32
	API          uint32
	Data         []byte
	Type         uint8
	handleStatus chan *MessageBase
	buf          []byte
}

func (m *MessageBase) GetId() uint32 {
	return m.Id
}

func (m *MessageBase) GetAPI() uint32 {
	return m.API
}

func (m *MessageBase) GetData() []byte {
	return m.Data
}

func (m *MessageBase) GetType() uint8 {
	return m.Type
}

func (m *MessageBase) String() string {
	return fmt.Sprintf("message >> id [%d] api [%d] data %v type [%s]", m.Id, m.API, m.Data, MessageBaseTypeMap[m.Type])
}

func (m *MessageBase) debug() {
	fmt.Println(m.String())
}
func (m *MessageBase) Marshal() []byte {
	buf, err := jeans.BaseTypeToBytes(m.Id, m.API, m.Type, m.Data)
	if err != nil {
		panic(any(err))
	}
	return buf
}

func (m *MessageBase) Unmarshal(b []byte) {
	m.buf = b
	err := jeans.BytesToBaseType(b, &m.Id, &m.API, &m.Type, &m.Data)
	if err != nil {
		panic(any(err))
	}
}

func (m *MessageBase) get() *MessageBase {
	return m
}

var messageBaseId uint32

func newMessageBase(api uint32, _type uint8, data []byte) *MessageBase {
	return &MessageBase{
		Id:           atomic.AddUint32(&messageBaseId, DEFAULT_messageBaseIdStep),
		API:          api,
		Type:         _type,
		Data:         data,
		handleStatus: make(chan *MessageBase),
	}
}

func NewMessageBase(api uint32, _type uint8, data []byte) *MessageBase {
	return newMessageBase(api, _type, data)
}

func NewMessageBaseWithUnmarshal(b []byte) *MessageBase {
	msg := new(MessageBase)
	msg.handleStatus = make(chan *MessageBase)
	msg.Unmarshal(b)
	return msg
}

type MessageForward struct {
	SrcId  string
	DestId string
	Data   []byte
}

func NewMessageForward(srcId, destId string, data []byte) *MessageForward {
	msg := new(MessageForward)
	msg.SrcId = srcId
	msg.DestId = destId
	msg.Data = data
	return msg
}

func (m *MessageForward) Marshal() []byte {
	buf, err := jeans.BaseTypeToBytes(m.SrcId, m.DestId, m.Data)
	if err != nil {
		panic(any(err))
	}
	return buf
}

func (m *MessageForward) Unmarshal(buf []byte) {
	err := jeans.BytesToBaseType(buf, &m.SrcId, &m.DestId, &m.Data)
	if err != nil {
		panic(any(err))
	}
}

func (m *MessageForward) String() string {
	return fmt.Sprintf("srcid :%s distid: %s data: %s", m.SrcId, m.DestId, m.Data)
}

func NewMessageForwardWithUnmarshal(b []byte) *MessageForward {
	msg := new(MessageForward)
	msg.Unmarshal(b)
	return msg
}

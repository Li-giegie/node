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
	return fmt.Sprintf("message >> id [%v] api [%v] data %v type [%v]", m.Id, m.API, m.Data, MessageBaseTypeMap[m.Type])
}

func (m *MessageBase) debug() {
	fmt.Println(m.String())
}
func (m *MessageBase) Marshal() ([]byte, error) {
	return jeans.BaseTypeToBytes(m.Id, m.API, m.Type, m.Data)
}

func (m *MessageBase) Unmarshal(b []byte) error {
	m.buf = b
	return jeans.BytesToBaseType(b, &m.Id, &m.API, &m.Type, &m.Data)
}

func (m *MessageBase) get() *MessageBase {
	return m
}

var messageBaseId uint32

func newMessageBase(id, api uint32, _type uint8, data []byte) *MessageBase {
	return &MessageBase{
		Id:           id,
		API:          api,
		Type:         _type,
		Data:         data,
		handleStatus: make(chan *MessageBase),
	}
}

func NewMessageBase(api uint32, _type uint8, data []byte) *MessageBase {
	id := atomic.AddUint32(&messageBaseId, DEFAULT_messageBaseIdStep)
	return newMessageBase(id, api, _type, data)
}

func NewMessageBaseWithId(id uint32, api uint32, _type uint8, data []byte) *MessageBase {
	return newMessageBase(id, api, _type, data)
}

func NewMessageBaseWithDataStr(api uint32, _type uint8, data string) *MessageBase {
	id := atomic.AddUint32(&messageBaseId, DEFAULT_messageBaseIdStep)
	return newMessageBase(id, api, _type, []byte(data))
}

func NewMessageBaseWithUnmarshal(b []byte) (*MessageBase, error) {
	msg := new(MessageBase)
	msg.handleStatus = make(chan *MessageBase)
	err := msg.Unmarshal(b)
	return msg, err
}

package node

import (
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"sync"
	"sync/atomic"
)

const (
	//心跳消息
	msgType_Tick uint8 = iota
	msgType_TickResp

	//仅发送消息
	msgType_Send

	//请求消息：需要回复的消息
	msgType_Req
	msgType_RespFail
	msgType_RespSuccess

	//转发请求消息：转发后需要回复的消息
	msgType_Forward
	msgType_ForwardSuccess
	msgType_ForwardFail

	msgType_Registration
	msgType_RegistrationSucccess
	msgType_RegistrationFail
)

var msgTypeMap = map[uint8]string{
	msgType_Send: "msgType_Send",

	msgType_Req:         "msgType_Req",
	msgType_RespSuccess: "msgType_RespSuccess",
	msgType_RespFail:    "msgType_RespFail",

	msgType_Forward:        "msgType_Forward",
	msgType_ForwardSuccess: "msgType_Forward",
	msgType_ForwardFail:    "msgType_ForwardFail",

	msgType_Tick:     "Tick",
	msgType_TickResp: "TickRespOk",

	msgType_Registration:         "msgType_Registration",
	msgType_RegistrationSucccess: "msgType_RegistrationSucccess",
	msgType_RegistrationFail:     "msgType_RegistrationFail",
}

var msgCounter uint32

//var _msgPoolCount uint32

var msgPool = sync.Pool{New: func() any {
	return new(message)
}}

type message struct {
	id    uint32
	api   uint32
	typ   uint8
	srcId uint64
	dstId uint64
	data  []byte
}

func (m *message) recycle() {
	m.id = 0
	m.api = 0
	m.data = nil
	m.typ = 0
	m.srcId = 0
	m.dstId = 0
	msgPool.Put(m)
}

func (m *message) marshalV1() []byte {
	buf, err := jeans.Encode(m.typ, m.id, m.api, m.data, m.srcId, m.dstId)
	if err != nil {
		panic(any(err))
	}
	return buf
}

func (m *message) unmarshalV1(buf []byte) {
	err := jeans.Decode(buf, &m.typ, &m.id, &m.api, &m.data, &m.srcId, &m.dstId)
	if err != nil {
		panic(any(err))
	}
}

func newMsgWithUnmarshalV1(b []byte) *message {
	m := msgPool.Get().(*message)
	m.unmarshalV1(b)
	return m
}

// newMsgWithReq 新建一个关于请求的数据包
func newMsgWithReq(api uint32, data []byte) *message {
	m := msgPool.Get().(*message)
	m.id = atomic.AddUint32(&msgCounter, 1)
	m.typ = msgType_Req
	m.api = api
	m.data = data
	return m
}

func newMsgWithForward(srcId, destId uint64, api uint32, data []byte) *message {
	m := msgPool.Get().(*message)
	m.id = atomic.AddUint32(&msgCounter, 1)
	m.typ = msgType_Forward
	m.srcId = srcId
	m.dstId = destId
	m.api = api
	m.data = data
	return m
}

// newMsgWithReq 新建一个关于请求的数据包
func newMsgWithTick() *message {
	m := msgPool.Get().(*message)
	m.id = atomic.AddUint32(&msgCounter, 1)
	m.typ = msgType_Tick
	return m
}

// msgType_Send 新建一个关于单次发送的数据包包
func newMsgWithSend(api uint32, data []byte) *message {
	m := msgPool.Get().(*message)
	m.id = atomic.AddUint32(&msgCounter, 1)
	m.typ = msgType_Send
	m.api = api
	m.data = data
	return m
}

func newMsgWithRegistration(apiList []uint32) *message {
	m := msgPool.Get().(*message)
	m.id = atomic.AddUint32(&msgCounter, 1)
	m.typ = msgType_Registration
	buf, err := jeans.EncodeSlice(apiList)
	if err != nil {
		panic(err)
	}
	m.data = buf
	return m
}

func newMsgWithRegistrationResp(m *message, ok bool, text string, badApiList []uint32) *message {
	if ok {
		m.typ = msgType_RegistrationSucccess
	} else {
		m.typ = msgType_RegistrationFail
	}
	buf, err := jeans.EncodeSlice(badApiList)
	if err != nil {
		panic(err)
	}
	buf, err = jeans.Encode(text, buf)
	if err != nil {
		panic(err)
	}
	m.data = buf
	return m
}

func newMsgWithUnmarshalV2(b []byte) *message {
	m := msgPool.Get().(*message)
	m.unmarshalV2(b)
	return m
}

func (m *message) marshalV2() []byte {
	switch m.typ {
	case msgType_Send:
		return m.marshalWithSend()
	case msgType_Req, msgType_RespSuccess, msgType_RespFail:
		return m.marshalWithReq()
	case msgType_Forward, msgType_ForwardSuccess, msgType_ForwardFail:
		return m.marshalWithForward()
	case msgType_Tick, msgType_TickResp:
		return m.marshalWithTick()
	case msgType_Registration, msgType_RegistrationSucccess, msgType_RegistrationFail:
		return m.marshalRegistration()
	default:
		panic("marshal err: msg type unlawfulness")
	}
}

func (m *message) unmarshalV2(buf []byte) {
	switch buf[0] {
	case msgType_Send:
		m.unmarshalWithSend(buf)
	case msgType_Req, msgType_RespSuccess, msgType_RespFail:
		m.unmarshalWithReq(buf)
	case msgType_Forward, msgType_ForwardSuccess, msgType_ForwardFail:
		m.unmarshalWithForward(buf)
	case msgType_Tick, msgType_TickResp:
		m.unmarshalWithTick(buf)
	case msgType_Registration, msgType_RegistrationSucccess, msgType_RegistrationFail:
		m.unmarshalRegistration(buf)
	default:
		panic("unmarshal err: msg type unlawfulness")
	}
}

func (m *message) marshalWithReq() []byte {
	buf, err := jeans.Encode(m.typ, m.id, m.api, m.data)
	if err != nil {
		panic(any(err))
	}
	return buf
}
func (m *message) unmarshalWithReq(buf []byte) {
	err := jeans.Decode(buf, &m.typ, &m.id, &m.api, &m.data)
	if err != nil {
		panic(any(err))
	}
}

func (m *message) marshalWithForward() []byte {
	buf, err := jeans.Encode(m.typ, m.id, m.api, m.srcId, m.dstId, m.data)
	if err != nil {
		panic(any(err))
	}
	return buf
}
func (m *message) unmarshalWithForward(buf []byte) {
	err := jeans.Decode(buf, &m.typ, &m.id, &m.api, &m.srcId, &m.dstId, &m.data)
	if err != nil {
		panic(any(err))
	}
}

func (m *message) marshalWithTick() []byte {
	buf, err := jeans.Encode(m.typ, m.id, m.data)
	if err != nil {
		panic(any(err))
	}
	return buf
}
func (m *message) unmarshalWithTick(buf []byte) {
	err := jeans.Decode(buf, &m.typ, &m.id, &m.data)
	if err != nil {
		panic(any(err))
	}
}

func (m *message) marshalWithSend() []byte {
	buf, err := jeans.Encode(m.typ, m.id, m.api, m.data)
	if err != nil {
		panic(any(err))
	}
	return buf
}
func (m *message) unmarshalWithSend(buf []byte) {
	err := jeans.Decode(buf, &m.typ, &m.id, &m.api, &m.data)
	if err != nil {
		panic(any(err))
	}
}

func (m *message) marshalRegistration() []byte {
	buf, err := jeans.Encode(m.typ, m.id, m.data)
	if err != nil {
		panic(any("err1 " + err.Error()))
	}
	return buf
}

func (m *message) unmarshalRegistration(buf []byte) {
	err := jeans.Decode(buf, &m.typ, &m.id, &m.data)
	if err != nil {
		panic(any("err2 " + err.Error()))
	}
}

func (m *message) String() string {
	if m == nil {
		return "nil"
	}
	return fmt.Sprintf("message : id [%d] api [%d] type [%s] data %s", m.id, m.api, msgTypeMap[m.typ], m.data)
}

func (m *message) debug() {
	fmt.Println(m.String())
}

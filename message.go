package node

import (
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"sync"
	"sync/atomic"
)

const (
	//心跳消息
	MsgType_Tick uint8 = iota
	MsgType_TickResp

	//仅发送消息
	MsgType_Send

	//请求消息：需要回复的消息
	MsgType_Req
	MsgType_RespFail
	MsgType_RespSuccess

	//转发请求消息：转发后需要回复的消息
	MsgType_ReqForward
	//MsgType_ReqForwardFail
	MsgType_RespForwardSuccess
	MsgType_RespForwardFail

	//服务端请求消息
	MsgType_ServerReq
	MsgType_ServerRespFail
	MsgType_ServerRespSuccess

	msg_type_req = 100 + iota
	msg_type_forward
	msg_type_tick
	msg_type_server_req
	msg_type_send
)

var MsgTypeMap = map[uint8]string{
	MsgType_Req:         "MsgType_Req",
	MsgType_RespSuccess: "MsgType_RespSuccess",
	MsgType_RespFail:    "MsgType_RespFail",

	MsgType_Send: "MsgType_Send",

	MsgType_ReqForward:         "MsgType_ReqForward",
	MsgType_RespForwardSuccess: "MsgType_RespForward",
	MsgType_RespForwardFail:    "MsgType_RespForwardFail",

	MsgType_Tick:     "Tick",
	MsgType_TickResp: "TickRespOk",

	MsgType_ServerReq:         "MsgType_ServerReq",
	MsgType_ServerRespSuccess: "MsgType_ServerRespSuccess",
	MsgType_ServerRespFail:    "MsgType_ServerRespFail",
}

var msgCounter uint32

//var _msgPoolCount uint32

var msgPool = sync.Pool{New: func() any {
	return new(message)
}}

type message struct {
	id    uint32
	API   uint32
	_type uint8
	Data  []byte
	srcId uint64
	dstId uint64
}

func (m *message) recycle() {
	m.id = 0
	m.API = 0
	m.Data = nil
	m._type = 0
	m.srcId = 0
	m.dstId = 0
	msgPool.Put(m)
}

func (m *message) marshalV1() []byte {
	buf, err := jeans.Encode(m._type, m.id, m.API, m.Data, m.srcId, m.dstId)
	if err != nil {
		panic(any(err))
	}
	return buf
}

func (m *message) unmarshalV1(buf []byte) {
	err := jeans.Decode(buf, &m._type, &m.id, &m.API, &m.Data, &m.srcId, &m.dstId)
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
	m._type = MsgType_Req
	m.API = api
	m.Data = data
	return m
}

func newMsgWithServReq(api uint32, data []byte) *message {
	m := msgPool.Get().(*message)
	m.id = atomic.AddUint32(&msgCounter, 1)
	m._type = MsgType_ServerReq
	m.API = api
	m.Data = data
	return m
}

func newMsgWithForward(srcId, destId uint64, api uint32, data []byte) *message {
	m := msgPool.Get().(*message)
	m.id = atomic.AddUint32(&msgCounter, 1)
	m._type = MsgType_ReqForward
	m.srcId = srcId
	m.dstId = destId
	m.API = api
	m.Data = data
	return m
}

// newMsgWithReq 新建一个关于请求的数据包
func newMsgWithTick() *message {
	m := msgPool.Get().(*message)
	m.id = atomic.AddUint32(&msgCounter, 1)
	m._type = MsgType_Tick
	return m
}

// MsgType_Send 新建一个关于单次发送的数据包包
func newMsgWithSend(api uint32, data []byte) *message {
	m := msgPool.Get().(*message)
	m.id = atomic.AddUint32(&msgCounter, 1)
	m._type = MsgType_Send
	m.API = api
	m.Data = data
	return m
}

func newMsgWithUnmarshalV2(b []byte) *message {
	m := msgPool.Get().(*message)
	m.unmarshalV2(b)
	return m
}

func (m *message) marshalV2() []byte {
	switch m.typ() {
	case msg_type_req:
		return m.marshalWithReq()
	case msg_type_forward:
		return m.marshalWithForward()
	case msg_type_tick:
		return m.marshalWithTick()
	case msg_type_server_req:
		return m.marshalWithSrvReq()
	case MsgType_Send:
		return m.marshalWithSend()
	default:
		panic("marshal err: msg type unlawfulness")
	}
}

func (m *message) unmarshalV2(buf []byte) {
	switch getMsgType(buf[0]) {
	case msg_type_req:
		m.unmarshalWithReq(buf)
	case msg_type_forward:
		m.unmarshalWithForward(buf)
	case msg_type_tick:
		m.unmarshalWithTick(buf)
	case msg_type_server_req:
		m.unmarshalWithSrvReq(buf)
	case MsgType_Send:
		m.unmarshalWithSrvSend(buf)
	default:
		panic("unmarshal err: msg type unlawfulness")
	}
}

func (m *message) marshalWithReq() []byte {
	buf, err := jeans.Encode(m._type, m.id, m.API, m.Data)
	if err != nil {
		panic(any(err))
	}
	return buf
}
func (m *message) unmarshalWithReq(buf []byte) {
	err := jeans.Decode(buf, &m._type, &m.id, &m.API, &m.Data)
	if err != nil {
		panic(any(err))
	}
}

func (m *message) marshalWithForward() []byte {
	buf, err := jeans.Encode(m._type, m.id, m.API, m.srcId, m.dstId, m.Data)
	if err != nil {
		panic(any(err))
	}
	return buf
}
func (m *message) unmarshalWithForward(buf []byte) {
	err := jeans.Decode(buf, &m._type, &m.id, &m.API, &m.srcId, &m.dstId, &m.Data)
	if err != nil {
		panic(any(err))
	}
}

func (m *message) marshalWithTick() []byte {
	buf, err := jeans.Encode(m._type, m.id, m.Data)
	if err != nil {
		panic(any(err))
	}
	return buf
}
func (m *message) unmarshalWithTick(buf []byte) {
	err := jeans.Decode(buf, &m._type, &m.id, &m.Data)
	if err != nil {
		panic(any(err))
	}
}

func (m *message) marshalWithSrvReq() []byte {
	buf, err := jeans.Encode(m._type, m.id, m.API, m.Data)
	if err != nil {
		panic(any(err))
	}
	return buf
}
func (m *message) unmarshalWithSrvReq(buf []byte) {
	err := jeans.Decode(buf, &m._type, &m.id, &m.API, &m.Data)
	if err != nil {
		panic(any(err))
	}
}

func (m *message) marshalWithSend() []byte {
	buf, err := jeans.Encode(m._type, m.id, m.API, m.Data)
	if err != nil {
		panic(any(err))
	}
	return buf
}
func (m *message) unmarshalWithSrvSend(buf []byte) {
	err := jeans.Decode(buf, &m._type, &m.id, &m.API, &m.Data)
	if err != nil {
		panic(any(err))
	}
}

func (m *message) typ() uint8 {
	return getMsgType(m._type)
}

func getMsgType(typ uint8) uint8 {
	switch typ {
	case MsgType_Req, MsgType_RespFail, MsgType_RespSuccess:
		return msg_type_req
	case MsgType_ReqForward, MsgType_RespForwardSuccess, MsgType_RespForwardFail:
		return msg_type_forward
	case MsgType_Tick, MsgType_TickResp:
		return msg_type_tick
	case MsgType_ServerReq, MsgType_ServerRespSuccess, MsgType_ServerRespFail:
		return msg_type_server_req
	case MsgType_Send:
		return MsgType_Send
	default:
		panic("parse data packet fail: Illegal data")
	}
}

func (m *message) String() string {
	if m == nil {
		return "nil"
	}
	return fmt.Sprintf("message : id [%d] api [%d] type [%s] data %s", m.id, m.API, MsgTypeMap[m._type], m.Data)
}

func (m *message) debug() {
	fmt.Println(m.String())
}

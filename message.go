package node

import (
	"errors"
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
	msgType_Reply
	msgType_ReplyErr

	msgType_Registration
	msgType_RegistrationResp
)

var msgTypeMap = map[uint8]string{
	msgType_Send:     "msgType_Send",
	msgType_Reply:    "msgType_Reply",
	msgType_ReplyErr: "msgType_ReplyErr",
	msgType_Tick:     "Tick",
	msgType_TickResp: "TickRespOk",

	msgType_Registration:     "msgType_Registration",
	msgType_RegistrationResp: "msgType_RegistrationResp",
}

var msgCounter uint32

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

func (m *message) marshal() []byte {
	buf, err := jeans.Encode(m.typ, m.id, m.api, m.data, m.srcId, m.dstId)
	if err != nil {
		panic(any(err))
	}
	return buf
}

func (m *message) unmarshal(buf []byte) {
	err := jeans.Decode(buf, &m.typ, &m.id, &m.api, &m.data, &m.srcId, &m.dstId)
	if err != nil {
		panic(any(err))
	}
}

func unmarshalMsg(b []byte) *message {
	m := msgPool.Get().(*message)
	m.unmarshal(b)
	return m
}

func newMsg() *message {
	m := msgPool.Get().(*message)
	m.id = atomic.AddUint32(&msgCounter, 1)
	return m
}

// newMsgWithReq 新建一个关于请求的数据包
func newMsgWithTick() *message {
	m := msgPool.Get().(*message)
	m.typ = msgType_Tick
	return m
}

func newMsgWithRegistration(apiList []uint32) *message {
	m := msgPool.Get().(*message)
	m.typ = msgType_Registration
	buf, err := jeans.EncodeSlice(apiList)
	if err != nil {
		panic(err)
	}
	m.data = buf
	return m
}

type RegistrationRespMsg struct {
	badApis []uint32
	err     error
}

func (m *message) String() string {
	if m == nil {
		return "nil"
	}
	return fmt.Sprintf("message : id [%d] api [%d] src [%d] dst [%d] type [%s] data %s", m.id, m.api, m.srcId, m.dstId, msgTypeMap[m.typ], m.data)
}

func (m *message) debug() {
	fmt.Println(m.String())
}

type msgDataWithErr struct {
	errStr string
	data   []byte
}

func (m *msgDataWithErr) Error() string {
	return m.errStr
}
func newMsgDataWithErr(m *message) *msgDataWithErr {
	mde := new(msgDataWithErr)
	err := jeans.Decode(m.data, &mde.errStr, &mde.data)
	if err != nil {
		panic(err)
	}
	return mde
}

func encodeErrReplyMsgData(err error, data []byte) []byte {
	buf, err := jeans.Encode(err.Error(), data)
	if err != nil {
		panic(err)
	}
	return buf
}

func encodeAuthReq(id uint64, data []byte) []byte {
	buf, err := jeans.Encode(id, data)
	if err != nil {
		panic(err)
	}
	return buf
}

func decodeAuthReq(buf []byte) (id uint64, data []byte) {
	err := jeans.Decode(buf, &id, &data)
	if err != nil {
		panic(err)
	}
	return
}

func encodeAuthResp(data []byte, err error) []byte {
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	buf, err := jeans.Encode(data, errStr)
	if err != nil {
		panic(err)
	}
	return buf
}

func decodeAuthResp(buf []byte) (data []byte, err error) {
	var errStr string
	err2 := jeans.Decode(buf, &data, &errStr)
	if err2 != nil {
		panic(err2)
	}
	if errStr != "" {
		err = errors.New(errStr)
	}
	return
}

func encodeRegistrationApiReq(apis []uint32) []byte {
	buf, err := jeans.EncodeSlice(apis)
	if err != nil {
		panic(err)
	}
	return buf
}

func decodeRegistrationApiReq(data []byte) (apis []uint32) {
	err := jeans.DecodeSlice(data, &apis)
	if err != nil {
		panic(err)
	}
	return
}

func encodeRegistrationApiResp(_err error, badApi []uint32) []byte {
	buf, err := jeans.EncodeSlice(badApi)
	if err != nil {
		panic(err)
	}
	var errStr string
	if _err != nil {
		errStr = _err.Error()
	}
	data, err := jeans.Encode(errStr, buf)
	if err != nil {
		panic(err)
	}
	return data
}

func decodeRegistrationApiResp(data []byte) (badApis []uint32, err error) {
	var errStr string
	var buf []byte
	if _err := jeans.Decode(data, &errStr, &buf); _err != nil {
		panic(_err)
	}
	if _err := jeans.DecodeSlice(buf, &badApis); _err != nil {
		panic(_err)
	}
	if errStr != "" {
		err = errors.New(errStr)
	}
	return
}

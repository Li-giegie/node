package node

import (
	"errors"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"io"
	"log"
	"math"
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
	msgType_Send:             "msgType_Send",
	msgType_Reply:            "msgType_Reply",
	msgType_ReplyErr:         "msgType_ReplyErr",
	msgType_Tick:             "Tick",
	msgType_TickResp:         "TickRespOk",
	msgType_Registration:     "msgType_Registration",
	msgType_RegistrationResp: "msgType_RegistrationResp",
}

var msgCounter uint32

var msgPool = sync.Pool{New: func() any {
	return new(message)
}}

const msg_headerLen = 29 //id+api+typ+src+dst+dataLen

type header struct {
	id    uint32
	api   uint32
	typ   uint8
	srcId uint64
	dstId uint64
}

func (m *header) marshal(dataLen uint32) []byte {
	buf, err := jeans.Encode(m.srcId, m.dstId, m.typ, m.id, m.api, dataLen)
	if err != nil {
		panic(any(err))
	}
	return buf
}

func (m *header) unmarshal(buf []byte) (dataLen uint32) {
	err := jeans.Decode(buf, &m.srcId, &m.dstId, &m.typ, &m.id, &m.api, &dataLen)
	if err != nil {
		panic(any(err))
	}
	return
}

type message struct {
	header
	data []byte
}

func (m *message) reply(typ uint8, data []byte) {
	m.header.typ = typ
	m.data = data
}

func (m *message) replyErr(typ uint8, data []byte, err error) {
	buf, _ := jeans.Encode(err.Error(), data)
	m.reply(typ, buf)
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
	return append(m.header.marshal(uint32(len(m.data))), m.data...)
}

func (m *message) unmarshal(buf []byte) {
	m.data = buf
}

func unmarshalMsg(b []byte) *message {
	m := msgPool.Get().(*message)
	m.unmarshal(b)
	return m
}

func newMsg(srcId, dstId uint64, typ uint8, api uint32, data []byte) *message {
	m := msgPool.Get().(*message)
	m.id = atomic.AddUint32(&msgCounter, 1)
	m.srcId = srcId
	m.dstId = dstId
	m.typ = typ
	m.api = api
	m.data = data
	return m
}

func (m *message) String() string {
	if m == nil {
		return "nil"
	}
	return fmt.Sprintf("message : id [%d] api [%d] src [%d] dst [%d] type [%s] data %s", m.id, m.api, m.srcId, m.dstId, msgTypeMap[m.typ], m.data)
}

func (m *message) debug() {
	log.Println(m.String())
}

func encodeErrReplyMsgData(_err error, data []byte) []byte {
	var buf []byte
	var err error
	if _err != nil {
		buf, err = jeans.Encode(_err.Error(), data)
	} else {
		buf, err = jeans.Encode("", data)
	}
	if err != nil {
		panic(err)
	}
	return buf
}

func decodeErrReplyMsgData(data []byte) ([]byte, error) {
	var errStr string
	var buf []byte
	if err := jeans.Decode(data, &errStr, &buf); err != nil {
		panic(err)
	}
	if errStr == "" {
		return buf, nil
	}
	return buf, errors.New(errStr)
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

const Version uint8 = 1

const auth_headerLen = 21 //version+srcid+dstid+datalen

type authMsg struct {
	version uint8
	srcId   uint64
	dstId   uint64
	data    []byte
}

func newAuthMsg(srcId, dstId uint64, data []byte) *authMsg {
	if len(data) > math.MaxUint16 {
		panic("auth data length > 65535")
	}
	return &authMsg{
		version: Version,
		srcId:   srcId,
		dstId:   dstId,
		data:    data,
	}
}

func (a *authMsg) marshal() []byte {
	buf, err := jeans.Encode(a.version, a.srcId, a.dstId, a.data)
	if err != nil {
		panic(err)
	}
	return buf
}
func (a *authMsg) unmarshal(conn io.Reader) error {
	buf, err := readAtLeast(conn, auth_headerLen)
	if err != nil {
		return fmt.Errorf("%v -0", ErrInvalidConnect)
	}
	var dataLen uint16
	if err = jeans.Decode(buf, &a.version, &a.srcId, &a.dstId, &dataLen); err != nil {
		return fmt.Errorf("%v -1", ErrInvalidConnect)
	}
	if dataLen > 0 {
		a.data, err = readAtLeast(conn, int(dataLen))
		if err != nil {
			return err
		}
	}
	return nil
}

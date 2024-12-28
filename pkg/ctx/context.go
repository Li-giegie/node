package ctx

import "github.com/Li-giegie/node/pkg/conn"

type Context interface {
	Id() uint32
	Type() uint8
	Hop() uint8
	SrcId() uint32
	DestId() uint32
	Data() []byte
	// Conn 当前上下文的连接
	Conn() conn.Conn
	// String 消息的字符串表达形式
	String() string
	// Response 响应请求，每个请求只能相应一次，如果发起端发送的是一个请求，不管使用什么type（protocol）响应类型必须是MsgType_Reply
	Response(code int16, data []byte) error
	// IsResponse 是否响应过
	IsResponse() bool
}

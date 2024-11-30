package iface

type Context interface {
	Id() uint32
	Type() uint8
	Hop() uint8
	SrcId() uint32
	DestId() uint32
	Data() []byte
	Conn() Conn
	// String 消息的字符串表达形式
	String() string
	// Reply 回复内容，每次请求限制回复一次，不要尝试多次回复，多次回复返回 OnceErr = errors.New("write only")
	Reply(data []byte) error
	// ReplyError 回复内容，每次请求限制回复一次，err 的长度限制 (err.Error()) 长度限制 math.MaxUint16-2 (65533)
	ReplyError(err error, data []byte) error
	// Stop 停止执行之后的回调
	Stop()
}

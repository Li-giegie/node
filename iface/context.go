package iface

type Context interface {
	Id() uint32
	Type() uint8
	SrcId() uint32
	DestId() uint32
	Data() []byte
	// String 消息的字符串表达形式
	String() string
	// Reply 回复内容，每次请求限制回复一次，不要尝试多次回复，多次回复返回 OnceErr = errors.New("write only")
	Reply(data []byte) error
	// ErrReply 回复内容，每次请求限制回复一次，err 的长度限制 (err.Error()) 长度限制 math.MaxUint16-2 (65533)
	ErrReply(data []byte, err error) error
	// CustomReply 回复内容，每次请求限制回复一次，自定义类型回复，适用需要修改消息类型的自定义发送的消息
	CustomReply(typ uint8, data []byte) error
	// Stop 停止执行之后的回调
	Stop()
	// Next 返回是否可以继续执行
	Next() bool
}

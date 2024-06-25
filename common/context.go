package common

import "github.com/Li-giegie/node/utils"

type Context interface {
	Id() uint32
	Type() uint8
	SrcId() uint16
	DestId() uint16
	Data() []byte
	String() string
	// Reply 回复内容，限制回复一次，不要尝试多次回复，多次回复返回 OnceErr = errors.New("write only")
	Reply(data []byte) error
	// ErrReply 回复内容，限制回复一次，不允许返回非空但为空字符串的err
	ErrReply(data []byte, err error) error
	// CustomReply 回复内容，限制回复一次，自定义类型回复，适用需要修改消息类型的自定义发送的消息
	CustomReply(typ uint8, data []byte) error
}

type WriterMsg interface {
	WriteMsg(m *Message) (err error)
}

type context struct {
	*Message
	WriterMsg
	once bool
}

func (c *context) Id() uint32 {
	return c.Message.Id
}

func (c *context) Type() uint8 {
	return c.Message.Type
}

func (c *context) SrcId() uint16 {
	return c.Message.SrcId
}

func (c *context) DestId() uint16 {
	return c.Message.DestId
}

func (c *context) Data() []byte {
	return c.Message.Data
}

// Reply 响应内容，限制回复一次，不要尝试多次回复，多次回复返回 OnceErr = errors.New("write only")
func (c *context) Reply(data []byte) error {
	if c.once {
		return DEFAULT_ErrMultipleReply
	}
	c.once = true
	return c.WriteMsg(c.Message.Reply(MsgType_Reply, data))
}

func (c *context) ErrReply(data []byte, err error) error {
	if c.once {
		return DEFAULT_ErrMultipleReply
	}
	c.once = true
	var errB []byte
	if err != nil {
		if errB = []byte(err.Error()); len(errB) == 0 {
			return DEFAULT_ErrReplyErrorInvalid
		}
	}
	return c.WriteMsg(c.Message.Reply(MsgType_ReplyErr, append(utils.PackBytes(errB), data...)))
}

func (c *context) CustomReply(typ uint8, data []byte) error {
	if c.once {
		return DEFAULT_ErrMultipleReply
	}
	c.once = true
	return c.WriteMsg(c.Message.Reply(typ, data))
}

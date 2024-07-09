package common

import (
	"encoding/binary"
)

type Context interface {
	Id() uint32
	Type() uint8
	SrcId() uint16
	DestId() uint16
	Data() []byte
	String() string
	// Reply 回复内容，每次请求限制回复一次，不要尝试多次回复，多次回复返回 OnceErr = errors.New("write only")
	Reply(data []byte) error
	// ErrReply 回复内容，每次请求限制回复一次，err 的长度限制 (err.Error()) 长度限制 math.MaxUint16-2 (65533)
	ErrReply(data []byte, err error) error
	// CustomReply 回复内容，每次请求限制回复一次，自定义类型回复，适用需要修改消息类型的自定义发送的消息
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

// ErrReply err length (uint16) <= 65533
func (c *context) ErrReply(data []byte, err error) error {
	if c.once {
		return DEFAULT_ErrMultipleReply
	}
	c.once = true
	var errB = make([]byte, 2)
	if err == nil {
		errB[0], errB[1] = 255, 255 //65535
	} else {
		errB2 := []byte(err.Error())
		errB2L := len(errB2)
		if errB2L > limitErrLen {
			return DEFAULT_ErrReplyErrorInvalid
		}
		errB = make([]byte, 2, 2+errB2L)
		binary.LittleEndian.PutUint16(errB, uint16(errB2L))
		errB = append(errB, errB2...)
	}
	return c.WriteMsg(c.Message.Reply(MsgType_ReplyErr, append(errB, data...)))
}

func (c *context) CustomReply(typ uint8, data []byte) error {
	if c.once {
		return DEFAULT_ErrMultipleReply
	}
	c.once = true
	return c.WriteMsg(c.Message.Reply(typ, data))
}

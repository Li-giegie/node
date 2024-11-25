package net

import "github.com/Li-giegie/node/message"

type Context struct {
	*message.Message
	*Connect
	once bool
	next bool
}

func (c *Context) Type() uint8 {
	return c.Message.Type
}

func (c *Context) Hop() uint8 {
	return c.Message.Hop
}

func (c *Context) Id() uint32 {
	return c.Message.Id
}

func (c *Context) SrcId() uint32 {
	return c.Message.SrcId
}

func (c *Context) DestId() uint32 {
	return c.Message.DestId
}

func (c *Context) Data() []byte {
	return c.Message.Data
}

func NewContext(connect *Connect, message *message.Message, next bool) *Context {
	return &Context{
		Message: message,
		Connect: connect,
		next:    next,
	}
}

// Reply 响应内容，限制回复一次，不要尝试多次回复，多次回复返回 var ErrLimitReply = errors.New("limit reply to one time")
func (c *Context) Reply(data []byte) (err error) {
	return c.CustomReply(message.MsgType_Reply, data)
}

// ErrReply error参数最大传输字节，65535代表error为nil，65534被保留不可用
const maxErrReplySize = 65533

// ErrReply err length <= 65533 byte
func (c *Context) ErrReply(data []byte, err error) error {
	var errB = make([]byte, 2)
	if err == nil {
		errB[0], errB[1] = 255, 255 //65535
	} else {
		errBytes := []byte(err.Error())
		if len(errBytes) > maxErrReplySize {
			return DEFAULT_ErrReplyErrorLengthOverflow
		}
		errB[0], errB[1] = byte(len(errBytes)), byte(len(errBytes)>>8)
		errB = append(errB, errBytes...)
	}
	return c.CustomReply(message.MsgType_ReplyErr, append(errB, data...))
}

func (c *Context) CustomReply(typ uint8, data []byte) (err error) {
	if c.once {
		return DEFAULT_ErrReplyLimitOnce
	}
	c.once = true
	c.Message.Hop = 0
	c.Message.Type = typ
	c.Message.SrcId, c.Message.DestId = c.Message.DestId, c.Message.SrcId
	c.Message.Data = data
	_, err = c.WriteMsg(c.Message)
	return err
}

// Next 获取是否进入下一个回调
func (c *Context) Next() bool {
	return c.next
}

// Stop 是否进入下一个回调
func (c *Context) Stop() {
	c.next = false
}

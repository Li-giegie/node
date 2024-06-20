package common

import "errors"

func NewContext(m *Message, w Writer) *Context {
	return &Context{
		m: m,
		w: w,
	}
}

type Context struct {
	m    *Message
	w    Writer
	once bool
}

func (c *Context) Id() uint32 {
	return c.m.Id
}

func (c *Context) Type() uint8 {
	return c.m.Typ
}

func (c *Context) SrcId() uint16 {
	return c.m.SrcId
}

func (c *Context) DestId() uint16 {
	return c.m.DestId
}

func (c *Context) Api() uint16 {
	return c.m.Api
}

func (c *Context) Data() []byte {
	return c.m.Data
}

func (c *Context) String() string {
	return c.m.String()
}

var OnceErr = errors.New("write only")

// Write 响应内容，限制回复一次，不要尝试多次回复，多次回复返回 OnceErr = errors.New("write only")
func (c *Context) Write(data []byte) (int, error) {
	if c.once {
		return 0, OnceErr
	}
	c.once = true
	c.m.Reply(MsgType_Reply, data)
	return c.w.WriteMsg(c.m)
}

type HandleFunc func(ctx *Context)

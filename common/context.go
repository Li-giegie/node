package common

type WriterMsg interface {
	WriteMsg(m *Message) (err error)
}

func NewContext(m *Message, w WriterMsg) *Context {
	return &Context{
		m: m,
		w: w,
	}
}

type Context struct {
	m    *Message
	w    WriterMsg
	once bool
}

func (c *Context) Id() uint32 {
	return c.m.Id
}

func (c *Context) Type() uint8 {
	return c.m.Type
}

func (c *Context) SrcId() uint16 {
	return c.m.SrcId
}

func (c *Context) DestId() uint16 {
	return c.m.DestId
}

func (c *Context) Data() []byte {
	return c.m.Data
}

func (c *Context) String() string {
	return c.m.String()
}

// Reply 响应内容，限制回复一次，不要尝试多次回复，多次回复返回 OnceErr = errors.New("write only")
func (c *Context) Reply(data []byte) error {
	if c.once {
		return DEFAULT_ErrMultipleReply
	}
	c.once = true
	return c.w.WriteMsg(c.m.Reply(MsgType_Reply, data))
}

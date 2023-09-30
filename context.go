package node

type HandlerFunc func(ctx *Context)

type Context struct {
	*Message
	write func(m *Message) error
}

func NewContext(msg *Message, write func(m *Message) error) *Context {
	ctx := new(Context)
	ctx.Message = msg
	ctx.write = write
	return ctx
}

func (c *Context) Write(b []byte) error {
	if c._type == MsgType_Req {
		c._type = MsgType_Resp
	} else if c._type == MsgType_ReqForward {
		c._type = MsgType_RespForward
	} else if c._type == MsgType_Tick {
		c._type = MsgType_TickResp
	}
	c.Data = b
	c.localId, c.remoteId = c.remoteId, c.localId
	return c.write(c.Message)
}

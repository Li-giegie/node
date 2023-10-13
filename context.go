package node

type HandlerFunc func(ctx *Context)

type HandleFunc func(data []byte) (out []byte, err error)

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
	var t uint8
	if c._type == MsgType_Req {
		t = MsgType_Resp
	} else if c._type == MsgType_ReqForward {
		t = MsgType_RespForward
	} else if c._type == MsgType_Tick {
		t = MsgType_TickResp
	}
	c._type = t
	c.Data = b
	c.localId, c.remoteId = c.remoteId, c.localId
	return c.write(c.Message)
}

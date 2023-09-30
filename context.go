package node

import "errors"

type HandlerFunc func(ctx *Context)

type ContextI interface {
	write(m *MessageBase) error
	Close()
}

type Context struct {
	ContextI
	MessageBaseI
}

func NewContext(connCtx ContextI, msg *MessageBase) *Context {
	var ctx = new(Context)
	ctx.ContextI = connCtx
	ctx.MessageBaseI = msg
	return ctx
}

func (c *Context) Write(b []byte) error {
	m := c.get()

	m.Data = b
	return c.write(m)
	switch m.Type {
	case MessageBaseType_RequestForward:
		fm := NewMessageForwardWithUnmarshal(m.Data)
		fm.Data = b
		fm.SrcId, fm.DestId = fm.DestId, fm.SrcId
		m.Data = fm.Marshal()
		m.Type = MessageBaseType_ResponseForward
	case MessageBaseType_Request:
		m.Type = MessageBaseType_Response
	case MessageBaseType_Tick:
		m.Type = MessageBaseType_TickReply
	default:
		return errors.New("message type does not support write" + MessageBaseTypeMap[m.Type])
	}
	return c.write(m)
}

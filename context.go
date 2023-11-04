package node

type Context struct {
	*message
	write       func(m *message) error
	setRespChan func(key any) (value any, ok bool)
}

func NewContext(msg *message, write func(m *message) error) *Context {
	ctx := new(Context)
	ctx.message = msg
	ctx.write = write
	return ctx
}

package node

import (
	"fmt"
	utils "github.com/Li-giegie/go-utils"
	"os"
)

type iContext interface {
	writeMsg(m *message) error
}

type Context struct {
	*message
	iContext
}

func newContext(m *message, iCtx iContext) *Context {
	return &Context{
		message:  m,
		iContext: iCtx,
	}
}

func (c *Context) SrcId() uint64 {
	return c.srcId
}

func (c *Context) Api() uint32 {
	return c.api
}

func (c *Context) Data() []byte {
	return c.data
}

func (c *Context) Reply(data []byte) error {
	c.message.reply(msgType_Reply, data)
	return c.iContext.writeMsg(c.message)
}
func (c *Context) ReplyErr(err error, data []byte) error {
	c.message.replyErr(msgType_ReplyErr, data, err)
	return c.iContext.writeMsg(c.message)
}

type HandlerFunc func(ctx *Context)

type Handler struct {
	cache *utils.MapUint32
}

func NewHandler() *Handler {
	h := new(Handler)
	h.cache = utils.NewMapUint32()
	return h
}

func (h *Handler) Add(api uint32, handleFunc HandlerFunc) {
	if _, ok := h.cache.Get(api); ok {
		fmt.Printf("error: handle api [%d] exist\n", api)
		os.Exit(1)
	}
	h.cache.Set(api, handleFunc)
}

func (h *Handler) Get(api uint32) (HandlerFunc, bool) {
	_any, ok := h.cache.Get(api)
	if !ok {
		return nil, false
	}
	return _any.(HandlerFunc), true
}

func (h *Handler) Del(api uint32) {
	h.cache.Delete(api)
}

func (h *Handler) Range(f func(api uint32, ih HandlerFunc)) {
	h.cache.Range(func(k uint32, v interface{}) {
		f(k, v.(HandlerFunc))
	})
}

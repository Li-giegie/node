package node

import (
	"fmt"
	utils "github.com/Li-giegie/go-utils"
	"os"
)

type iContext interface {
	writeMsg(m *message) error
}

func newContext(m *message, iCtx iContext) *Context {
	return &Context{
		message:  m,
		iContext: iCtx,
	}
}

type Context struct {
	*message
	iContext
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

type iHandler interface {
	AddHandle(api uint32, handleFunc HandlerFunc)
	GetHandle(api uint32) (HandlerFunc, bool)
	DeleteHandle(api uint32)
	RangeHandle(f func(api uint32, ih HandlerFunc))
	HandlerKeys() []uint32
}

func newHandler() iHandler {
	h := new(handler)
	h.cache = utils.NewMapUint32()
	return h
}

type handler struct {
	cache *utils.MapUint32
}

func (h *handler) AddHandle(api uint32, handleFunc HandlerFunc) {
	if _, ok := h.cache.Get(api); ok {
		fmt.Printf("error: handle api [%d] exist\n", api)
		os.Exit(1)
	}
	h.cache.Set(api, handleFunc)
}

func (h *handler) GetHandle(api uint32) (HandlerFunc, bool) {
	i, ok := h.cache.Get(api)
	if !ok {
		return nil, false
	}
	return i.(HandlerFunc), true
}

func (h *handler) DeleteHandle(api uint32) {
	h.cache.Delete(api)
}

func (h *handler) HandlerKeys() []uint32 {
	return h.cache.KeyToSlice()
}

func (h *handler) RangeHandle(f func(api uint32, ih HandlerFunc)) {
	h.cache.Range(func(k uint32, v interface{}) {
		f(k, v.(HandlerFunc))
	})
}

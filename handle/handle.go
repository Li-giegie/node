package handle

import (
	"fmt"
	utils "github.com/Li-giegie/go-utils"
	"os"
)

const (
	HandlerTYpe_Send uint8 = 1 + iota
	HandlerTYpe_Request
)

type IHandler interface {
	Typ() uint8
}

type HandlerFuncSend func(id uint64, data []byte)

func (HandlerFuncSend) Typ() uint8 {
	return HandlerTYpe_Send
}

type HandlerFuncRequest func(id uint64, data []byte) ([]byte, error)

func (HandlerFuncRequest) Typ() uint8 {
	return HandlerTYpe_Request
}

type Handler struct {
	cache *utils.MapUint32
}

func NewHandler() *Handler {
	h := new(Handler)
	h.cache = utils.NewMapUint32()
	return h
}

func (h *Handler) Add(api uint32, handleFunc IHandler) {
	if _, ok := h.cache.Get(api); ok {
		fmt.Printf("error: handle api [%d] exist\n", api)
		os.Exit(1)
	}
	h.cache.Set(api, handleFunc)
}

func (h *Handler) Get(api uint32) (IHandler, bool) {
	_any, ok := h.cache.Get(api)
	if !ok {
		return nil, false
	}
	return _any.(IHandler), true
}

func (h *Handler) Del(api uint32) {
	h.cache.Delete(api)
}

func (h *Handler) Range(f func(api uint32, ih IHandler)) {
	h.cache.Range(func(k uint32, v interface{}) {
		f(k, v.(IHandler))
	})
}

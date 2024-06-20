package common

import (
	"sync"
)

type ServeMux struct {
	m map[uint16]HandleFunc
	l sync.RWMutex
}

func (h *ServeMux) HandleFunc(api uint16, f HandleFunc) bool {
	ok := false
	h.l.Lock()
	if _, ok = h.m[api]; !ok {
		h.m[api] = f
	}
	h.l.Unlock()
	return !ok
}

func (h *ServeMux) GetHandleFunc(api uint16) (HandleFunc, bool) {
	h.l.RLock()
	v, ok := h.m[api]
	h.l.RUnlock()
	return v, ok
}

func (h *ServeMux) Keys() []uint16 {
	h.l.RLock()
	apis := make([]uint16, 0, len(h.m))
	for u, _ := range h.m {
		apis = append(apis, u)
	}
	h.l.RUnlock()
	return apis
}

func NewServeMux() *ServeMux {
	return &ServeMux{
		m: make(map[uint16]HandleFunc),
		l: sync.RWMutex{},
	}
}

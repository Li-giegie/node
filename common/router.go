package common

import "sync"

// Router 路由
type Router interface {
	GetRouteNext(dst uint16) (v uint16, ok bool)
	DeleteRoute(dst uint16)
	AddRoute(dst, next uint16)
	AddRoutes(dst []uint16, next uint16)
	DeleteRoutes(dst []uint16)
	DeleteRouteNext(next uint16)
	DeleteRouteNextAll(next uint16)
}

type RouteTable struct {
	// map[dst]next
	cache map[uint16]uint16
	l     *sync.RWMutex
}

func NewRouter() *RouteTable {
	return &RouteTable{
		cache: make(map[uint16]uint16),
		l:     new(sync.RWMutex),
	}
}

func (r *RouteTable) GetRouteNext(dst uint16) (v uint16, ok bool) {
	r.l.RLock()
	v, ok = r.cache[dst]
	r.l.RUnlock()
	return
}

func (r *RouteTable) AddRoute(dst, next uint16) {
	r.l.Lock()
	r.cache[dst] = next
	r.l.Unlock()
}

func (r *RouteTable) AddRoutes(dst []uint16, next uint16) {
	r.l.Lock()
	for i := 0; i < len(dst); i++ {
		r.cache[dst[i]] = next
	}
	r.l.Unlock()
}

func (r *RouteTable) DeleteRoute(dst uint16) {
	r.l.Lock()
	delete(r.cache, dst)
	r.l.Unlock()
}

func (r *RouteTable) DeleteRoutes(dst []uint16) {
	r.l.Lock()
	for i := 0; i < len(dst); i++ {
		delete(r.cache, dst[i])
	}
	r.l.Unlock()
}

func (r *RouteTable) DeleteRouteNext(next uint16) {
	r.l.Lock()
	for u, u2 := range r.cache {
		if u2 == next {
			delete(r.cache, u)
			break
		}
	}
	r.l.Unlock()
}

func (r *RouteTable) DeleteRouteNextAll(next uint16) {
	r.l.Lock()
	for u, u2 := range r.cache {
		if u2 == next {
			delete(r.cache, u)
		}
	}
	r.l.Unlock()
}

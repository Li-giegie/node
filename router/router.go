package router

import (
	"sync"
)

func NewRouter() *Router {
	return &Router{
		cache: make(map[uint32]*RouteEmpty),
		l:     sync.RWMutex{},
	}
}

type Router struct {
	// dst --> paths
	cache map[uint32]*RouteEmpty
	l     sync.RWMutex
}

func (r *Router) AddRoute(dst, via uint32, hop uint8, unixNano int64, paths []*RoutePath) bool {
	r.l.Lock()
	defer r.l.Unlock()
	empty := r.cache[dst]
	if empty != nil && empty.UnixNano > unixNano {
		return false
	}
	r.cache[dst] = &RouteEmpty{
		Dst:      dst,
		Via:      via,
		Hop:      hop,
		UnixNano: unixNano,
		Paths:    paths,
	}
	return true
}

func (r *Router) RemoveRoute(dst uint32, unixNano int64) bool {
	r.l.Lock()
	defer r.l.Unlock()
	empty := r.cache[dst]
	if empty != nil && empty.UnixNano > unixNano {
		return false
	}
	delete(r.cache, dst)
	return true
}

func (r *Router) RemoveRouteWithVia(via uint32, unixNano int64) (n int) {
	r.l.Lock()
	defer r.l.Unlock()
	for u, empty := range r.cache {
		if empty.Via == via {
			for _, path := range empty.Paths {
				if path.Id == via {
					if unixNano >= path.UnixNano {
						delete(r.cache, u)
						n++
					}
					break
				}
			}
		}
	}
	return
}

func (r *Router) RemoveRouteWithPath(path uint32, unixNano int64) (n int) {
	r.l.Lock()
	defer r.l.Unlock()
	for u, empty := range r.cache {
		for _, routerPath := range empty.Paths {
			if routerPath.Id == path && unixNano >= routerPath.UnixNano {
				delete(r.cache, u)
				n++
				break
			}
		}
	}
	return
}

func (r *Router) GetRoute(dst uint32) (*RouteEmpty, bool) {
	r.l.RLock()
	defer r.l.RUnlock()
	empty, ok := r.cache[dst]
	return empty, ok
}

func (r *Router) GetRouteVia(dst uint32) (uint32, bool) {
	r.l.RLock()
	defer r.l.RUnlock()
	empty := r.cache[dst]
	if empty == nil {
		return 0, false
	}
	return empty.Via, true
}

func (r *Router) RangeRoute(callback func(*RouteEmpty) bool) {
	r.l.RLock()
	defer r.l.RUnlock()
	for _, empty := range r.cache {
		if !callback(empty) {
			return
		}
	}
}

type RouteEmpty struct {
	Dst      uint32
	Via      uint32
	Hop      uint8
	UnixNano int64
	Paths    []*RoutePath
}

type RoutePath struct {
	Id       uint32
	UnixNano int64
}

package routemanager

import (
	"github.com/Li-giegie/node/pkg/router"
	"sync"
)

func NewRouter() *Router {
	return &Router{
		cache: make(map[uint32]*router.RouteEmpty),
		l:     sync.RWMutex{},
	}
}

type Router struct {
	// dst --> RouteEmpty
	cache     map[uint32]*router.RouteEmpty
	l         sync.RWMutex
	rerouting []func(dst uint32) (*router.RouteEmpty, bool)
}

func (r *Router) AddRoute(dst, via uint32, hop uint8, unixNano int64, paths []uint32) bool {
	r.l.Lock()
	defer r.l.Unlock()
	if r.cache == nil {
		r.cache = make(map[uint32]*router.RouteEmpty)
	}
	empty := r.cache[dst]
	if empty == nil || hop <= empty.Hop {
		r.cache[dst] = &router.RouteEmpty{
			Dst:      dst,
			Via:      via,
			Hop:      hop,
			UnixNano: unixNano,
			Paths:    paths,
		}
		return true
	}
	return false
}

func (r *Router) RemoveRoute(dst uint32) bool {
	r.l.Lock()
	defer r.l.Unlock()
	empty := r.cache[dst]
	if empty != nil {
		delete(r.cache, dst)
		return true
	}
	return false
}

func (r *Router) RemoveRouteWithVia(via uint32) (n int) {
	r.l.Lock()
	defer r.l.Unlock()
	for u, empty := range r.cache {
		if empty.Via == via {
			delete(r.cache, u)
			n++
		}
	}
	return
}

func (r *Router) RemoveRouteWithPath(path uint32) (n int) {
	r.l.Lock()
	defer r.l.Unlock()
	for u, empty := range r.cache {
		for _, _path := range empty.Paths {
			if _path == path {
				delete(r.cache, u)
				n++
				break
			}
		}
	}
	return
}

func (r *Router) GetRoute(dst uint32) (*router.RouteEmpty, bool) {
	r.l.RLock()
	defer r.l.RUnlock()
	empty, ok := r.cache[dst]
	if !ok {
		return nil, false
	}
	return CopyRouteEmpty(empty), ok
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
func (r *Router) GetRouteDstWithVia(via uint32) []uint32 {
	r.l.RLock()
	defer r.l.RUnlock()
	var dsts []uint32
	for u, empty := range r.cache {
		if empty.Via == via {
			dsts = append(dsts, u)
		}
	}
	return dsts
}
func (r *Router) GetRouteDstWithPath(path uint32) []uint32 {
	r.l.RLock()
	defer r.l.RUnlock()
	var dsts []uint32
	for u, empty := range r.cache {
		for _, u2 := range empty.Paths {
			if u2 == path {
				dsts = append(dsts, u)
				break
			}
		}
	}
	return dsts
}

func (r *Router) RangeRoute(callback func(*router.RouteEmpty) bool) {
	r.l.RLock()
	defer r.l.RUnlock()
	for _, empty := range r.cache {
		if !callback(CopyRouteEmpty(empty)) {
			return
		}
	}
}

func CopyRouteEmpty(empty *router.RouteEmpty) *router.RouteEmpty {
	paths := make([]uint32, len(empty.Paths))
	copy(paths, empty.Paths)
	return &router.RouteEmpty{
		Dst:      empty.Dst,
		Via:      empty.Via,
		Hop:      empty.Hop,
		UnixNano: empty.UnixNano,
		Paths:    paths,
	}
}

func (r *Router) ReroutingHandleFunc(callback func(dst uint32) (*router.RouteEmpty, bool)) {
	r.rerouting = append(r.rerouting, callback)
}

func (r *Router) Rerouting(dst uint32) (empty *router.RouteEmpty, ok bool) {
	for _, f := range r.rerouting {
		if empty, ok = f(dst); ok {
			return
		}
	}
	return nil, false
}

func (r *Router) RouteLen() int {
	r.l.RLock()
	defer r.l.RUnlock()
	return len(r.cache)
}

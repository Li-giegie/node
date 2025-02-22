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

func (r *Router) AddRoute(dst, via uint32, hop uint8, unixNano int64, paths []*router.RoutePath) bool {
	r.l.Lock()
	defer r.l.Unlock()
	if r.cache == nil {
		r.cache = make(map[uint32]*router.RouteEmpty)
	}
	empty := r.cache[dst]
	if empty != nil && empty.UnixNano > unixNano {
		return false
	}
	r.cache[dst] = &router.RouteEmpty{
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

func (r *Router) GetRoute(dst uint32) (*router.RouteEmpty, bool) {
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

func (r *Router) RangeRoute(callback func(*router.RouteEmpty) bool) {
	r.l.RLock()
	defer r.l.RUnlock()
	for _, empty := range r.cache {
		if !callback(empty) {
			return
		}
	}
}

func (r *Router) ReroutingHandleFunc(callback func(dst uint32) (*router.RouteEmpty, bool)) {
	r.rerouting = append(r.rerouting, callback)
}

func (r *Router) Rerouting(dst uint32) (empty *router.RouteEmpty, ok bool) {
	for _, f := range r.rerouting {
		empty, ok = f(dst)
		if ok {
			return
		}
	}
	return nil, false
}

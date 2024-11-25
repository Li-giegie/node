package nodediscovery

import (
	"sync"
	"time"
)

func NewRouter() *Router {
	return &Router{
		cache: make(map[uint32]*RouteEmpty),
		l:     sync.RWMutex{},
	}
}

type RouteEmpty struct {
	dst      uint32
	via      uint32
	hop      uint8
	duration time.Duration
	fullPath []uint32
}

// Dst 路由目的
func (r *RouteEmpty) Dst() uint32 {
	return r.dst
}

// Via 路由下一跳
func (r *RouteEmpty) Via() uint32 {
	return r.via
}

// Hop 路由跳数，静态路由该值无效
func (r *RouteEmpty) Hop() uint8 {
	return r.hop
}

// Duration 路由更新时间 unixNano时间戳
func (r *RouteEmpty) Duration() time.Duration {
	return r.duration
}

// FullPath 从源到目的的路由路径
func (r *RouteEmpty) FullPath() []uint32 {
	return r.fullPath
}

type Router struct {
	cache map[uint32]*RouteEmpty
	l     sync.RWMutex
}

func (r *Router) AddRoute(dst, via uint32, hop uint8) {
	r.l.Lock()
	defer r.l.Unlock()
	r.cache[dst] = &RouteEmpty{
		dst:      dst,
		via:      via,
		hop:      hop,
		duration: time.Duration(time.Now().UnixNano()),
		fullPath: nil,
	}
}

func (r *Router) AddRouteWithFullPath(dst, via uint32, hop uint8, unixNao int64, fullPath []uint32) {
	r.l.Lock()
	defer r.l.Unlock()
	r.cache[dst] = &RouteEmpty{
		dst:      dst,
		via:      via,
		hop:      hop,
		duration: time.Duration(unixNao),
		fullPath: fullPath,
	}
}

func (r *Router) RemoveRoute(dst, via uint32) {
	r.l.Lock()
	defer r.l.Unlock()
	empty, ok := r.cache[dst]
	if !ok || empty.via != via {
		return
	}
	delete(r.cache, dst)
}

func (r *Router) RemoveRouteWithDst(dst uint32) {
	r.l.Lock()
	defer r.l.Unlock()
	delete(r.cache, dst)
}

func (r *Router) RemoveRouteWithVia(via uint32) {
	r.l.Lock()
	defer r.l.Unlock()
	for u, empty := range r.cache {
		if empty.via == via {
			delete(r.cache, u)
		}
	}
}

func (r *Router) RemoveRouteWithPath(path uint32) {
	r.l.Lock()
	defer r.l.Unlock()
	var exist bool
	for u, empty := range r.cache {
		exist = false
		for _, _path := range empty.fullPath {
			if _path == path {
				exist = true
				break
			}
		}
		if exist {
			delete(r.cache, u)
		}
	}
}

func (r *Router) GetRoute(dst uint32) (*RouteEmpty, bool) {
	r.l.RLock()
	defer r.l.RUnlock()
	empty, ok := r.cache[dst]
	return empty, ok
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

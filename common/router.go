package common

import (
	"sync"
	"time"
)

// Router 接口
type Router interface {
	AddRoute(dst, next, hop uint16)
	DeleteRouteNextHop(dst, next, hop uint16)
	DeleteRouteNext(dst, next uint16)
	DeleteRoute(dst uint16)
	// GetDstRoutes 获取下一条路由列表
	GetDstRoutes(dst uint16) []*Next
	// GetNextRoutes 获取通过下一跳能到达目的ID
	GetNextRoutes(next uint16) (dstId []uint16)
}

// RouteTable 路由表实现
type RouteTable struct {
	l *sync.RWMutex
	// cache dst 下一跳表
	cache map[uint16][]*Next
}

type Next struct {
	// Next 下一跳ID
	Next uint16
	// Hop 跳数 值越小间隔节点越少
	Hop uint16
	// UnixMill 更新时间
	UnixMill int64
}

func NewRouter() Router {
	return &RouteTable{
		l:     &sync.RWMutex{},
		cache: make(map[uint16][]*Next),
	}
}

func (r *RouteTable) AddRoute(dst, next, hop uint16) {
	newNext := &Next{
		Next:     next,
		Hop:      hop,
		UnixMill: time.Now().UnixMilli(),
	}
	r.l.Lock()
	nextList, ok := r.cache[dst]
	if !ok {
		r.cache[dst] = []*Next{newNext}
		r.l.Unlock()
		return
	}
	isAdd := false
	for i := 0; i < len(nextList); i++ {
		if newNext.Hop < nextList[i].Hop {
			nextList = append(nextList[:i], append([]*Next{newNext}, nextList[i:]...)...)
			isAdd = true
			break
		}
	}
	if !isAdd {
		nextList = append(nextList, newNext)
	}
	result := make([]*Next, 0, len(nextList))
	for i := 0; i < len(nextList); i++ {
		isAdd = true
		for j := 0; j < len(result); j++ {
			if result[j].Hop == nextList[i].Hop && result[j].Next == nextList[i].Next {
				isAdd = false
				break
			}
		}
		if isAdd {
			result = append(result, nextList[i])
		}
	}
	r.cache[dst] = result
	r.l.Unlock()
}

func (r *RouteTable) DeleteRouteNextHop(dst, next, hop uint16) {
	r.l.Lock()
	nextList, ok := r.cache[dst]
	if !ok {
		r.l.Unlock()
		return
	}
	newNext := make([]*Next, 0, len(nextList))
	for i := 0; i < len(nextList); i++ {
		if !(nextList[i].Next == next && nextList[i].Hop == hop) {
			newNext = append(newNext, nextList[i])
		}
	}
	r.cache[dst] = newNext
	r.l.Unlock()
}

func (r *RouteTable) DeleteRouteNext(dst, next uint16) {
	r.l.Lock()
	n, ok := r.cache[dst]
	if !ok {
		r.l.Unlock()
		return
	}
	newNext := make([]*Next, 0, len(n))
	for i := 0; i < len(n); i++ {
		if n[i].Next != next {
			newNext = append(newNext, n[i])
		}
	}
	r.cache[dst] = newNext
	r.l.Unlock()
}

func (r *RouteTable) DeleteRoute(dst uint16) {
	r.l.Lock()
	delete(r.cache, dst)
	r.l.Unlock()
}

func (r *RouteTable) GetNextRoutes(next uint16) []uint16 {
	r.l.RLock()
	var result []uint16
	var isAdd bool
	for u, nexts := range r.cache {
		isAdd = false
		for _, n := range nexts {
			if n.Next == next {
				isAdd = true
				break
			}
		}
		if isAdd {
			result = append(result, u)
		}
	}
	r.l.RUnlock()
	return result
}

func (r *RouteTable) GetDstRoutes(dst uint16) []*Next {
	r.l.RLock()
	res, _ := r.cache[dst]
	r.l.RUnlock()
	return res
}

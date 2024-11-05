package net

import (
	"sync"
)

type RouteTable struct {
	counter uint64
	tab     map[uint32]*RouteInfo
	l       sync.RWMutex
}

func NewRouteTable() *RouteTable {
	return &RouteTable{
		counter: 0,
		tab:     make(map[uint32]*RouteInfo),
		l:       sync.RWMutex{},
	}
}

type RouteInfo struct {
	id  uint64
	dst uint32
	via uint32
	hop uint8
}

func (route *RouteTable) AddRoute(dst, via uint32, hop uint8) (id uint64) {
	route.l.Lock()
	info, ok := route.tab[dst]
	if ok {
		if info.via != via && hop > info.hop {
			route.l.Unlock()
			return info.id
		}
	}
	route.counter++
	route.tab[dst] = &RouteInfo{
		id:  route.counter,
		dst: dst,
		via: via,
		hop: hop,
	}
	route.l.Unlock()
	return route.counter
}

func (route *RouteTable) RemoveRoute(dst, via uint32) (ok bool) {
	route.l.Lock()
	v, exist := route.tab[dst]
	if exist && v.via == via {
		delete(route.tab, dst)
		ok = true
	}
	route.l.Unlock()
	return
}

func (route *RouteTable) RemoveRouteWithDst(dst uint32) {
	route.l.Lock()
	delete(route.tab, dst)
	route.l.Unlock()
}

func (route *RouteTable) RemoveRouteWithVia(via uint32) (affected int) {
	route.l.Lock()
	for dst, info := range route.tab {
		if info.via == via {
			delete(route.tab, dst)
			affected++
		}
	}
	route.l.Unlock()
	return
}

func (route *RouteTable) RemoveRouteWithId(id uint64) bool {
	route.l.Lock()
	for dst, info := range route.tab {
		if info.id == id {
			delete(route.tab, dst)
			route.l.Unlock()
			return true
		}
	}
	route.l.Unlock()
	return false
}

func (route *RouteTable) GetRoute(dst uint32) (via uint32, exist bool) {
	route.l.RLock()
	info, ok := route.tab[dst]
	route.l.RUnlock()
	if ok {
		return info.via, true
	}
	return 0, false
}

func (route *RouteTable) GetRouteWithVia(via uint32) (dst []uint32) {
	route.l.RLock()
	dst = make([]uint32, 0, len(route.tab))
	for _, info := range route.tab {
		if info.via == via {
			dst = append(dst, info.dst)
		}
	}
	route.l.RUnlock()
	return dst
}

func (route *RouteTable) RangeRoute(f func(id uint64, dst uint32, via uint32, hop uint8)) {
	route.l.RLock()
	for _, info := range route.tab {
		f(info.id, info.dst, info.via, info.hop)
	}
	route.l.RUnlock()
}

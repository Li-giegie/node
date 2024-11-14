package net

import (
	"fmt"
	"github.com/Li-giegie/node/iface"
	"sync"
	"time"
)

type RouteTable struct {
	tab map[uint32]*RouteInfo
	l   sync.RWMutex
}

func NewRouteTable() *RouteTable {
	return &RouteTable{
		tab: make(map[uint32]*RouteInfo),
		l:   sync.RWMutex{},
	}
}

type RouteInfo struct {
	dst        uint32
	via        uint32
	hop        uint8
	activation time.Duration
}

func (r *RouteInfo) Dst() uint32 {
	return r.dst
}

func (r *RouteInfo) Via() uint32 {
	return r.via
}

func (r *RouteInfo) Hop() uint8 {
	return r.hop
}

func (r *RouteInfo) Activation() time.Duration {
	return r.activation
}

func (r *RouteInfo) String() string {
	return fmt.Sprintf("dst %d via %d hop %d activation %s", r.dst, r.via, r.hop, time.UnixMicro(r.activation.Microseconds()).Format("2006-01-02 15:04:05"))
}

func (route *RouteTable) AddRoute(dst, via uint32, hop uint8, d time.Duration) (isAdd bool) {
	route.l.Lock()
	info, ok := route.tab[dst]
	if ok && info.via != via && hop > info.hop {
		route.l.Unlock()
		return false
	}
	route.tab[dst] = &RouteInfo{
		dst:        dst,
		via:        via,
		hop:        hop,
		activation: d,
	}
	route.l.Unlock()
	return true
}

func (route *RouteTable) AddRouteWithCallback(dst, via uint32, hop uint8, d time.Duration, callback func(info iface.RouteInfo) (isAdd bool)) (isAdd bool) {
	route.l.Lock()
	info, ok := route.tab[dst]
	if ok {
		if callback(info) {
			info.via = via
			info.hop = hop
			info.activation = d
			isAdd = true
		} else {
			isAdd = false
		}
	} else {
		route.tab[dst] = &RouteInfo{
			dst:        dst,
			via:        via,
			hop:        hop,
			activation: d,
		}
		isAdd = true
	}
	route.l.Unlock()
	return isAdd
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

func (route *RouteTable) RemoveRouteWithCallback(dst uint32, callback func(info iface.RouteInfo) (isDel bool)) (isDel bool) {
	route.l.Lock()
	info, exist := route.tab[dst]
	if exist && callback(info) {
		delete(route.tab, dst)
		isDel = true
	}
	route.l.Unlock()
	return
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

func (route *RouteTable) GetRoute(dst uint32) (via uint32, hop uint8, exist bool) {
	route.l.RLock()
	info, ok := route.tab[dst]
	route.l.RUnlock()
	if ok {
		return info.via, info.hop, true
	}
	return 0, 0, false
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

func (route *RouteTable) RangeRoute(f func(info iface.RouteInfo)) {
	route.l.RLock()
	for _, info := range route.tab {
		f(info)
	}
	route.l.RUnlock()
}

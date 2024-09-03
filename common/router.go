package common

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

type Router interface {
	AddRoute(dst, next, hop, parentNode uint16)
	DeleteRoute(dst, next, hop, parentNode uint16) bool
	DeleteRouteAll(dst uint16)
	DeleteNextRoute(next uint16) bool
	RouteTableOutput() []byte
	GetDstRoutes(dst uint16) []*RouteInfo
}

type RouteTable struct {
	routes map[uint16][]*RouteInfo
	*sync.RWMutex
}

func NewRouter() *RouteTable {
	return &RouteTable{
		routes:  make(map[uint16][]*RouteInfo),
		RWMutex: &sync.RWMutex{},
	}
}

func (r *RouteTable) AddRoute(dst, next, hop, parentNode uint16) {
	route := &RouteInfo{
		UnixMilli:  time.Now().UnixMilli(),
		Next:       next,
		Hop:        hop,
		ParentNode: parentNode,
	}
	r.Lock()
	info, ok := r.routes[dst]
	if !ok {
		r.routes[dst] = []*RouteInfo{route}
		r.Unlock()
		return
	}
	index := searchRouteInfoHop(info, hop)
	checkIndex := index
	if checkIndex == -1 {
		checkIndex = len(info) - 1
	} else {
		checkIndex = index - 1
	}
	for i := checkIndex; i >= 0; i-- {
		if info[i].Hop != hop {
			break
		}
		if next == info[i].Next && parentNode == info[i].ParentNode {
			info[i].UnixMilli = time.Now().UnixMilli()
			r.routes[dst] = info
			r.Unlock()
			return
		}
	}
	if index == -1 {
		info = append(info, route)
	} else {
		info = append(info[:index], append([]*RouteInfo{route}, info[index:]...)...)
	}
	r.routes[dst] = info
	r.Unlock()
}

func (r *RouteTable) DeleteRoute(dst, next, hop, parentNode uint16) bool {
	r.Lock()
	info, ok := r.routes[dst]
	if !ok {
		r.Unlock()
		return false
	}
	index := searchRouteInfoNext(info, next)
	if index == -1 {
		r.Unlock()
		return false
	}
	delIndex := -1
	for i := index; i < len(info); i++ {
		if info[i].Next != next {
			break
		}
		if info[i].Hop == hop && info[i].ParentNode == parentNode {
			delIndex = i
			break
		}
	}
	if delIndex == -1 {
		r.Unlock()
		return false
	}
	if len(info) == 1 {
		delete(r.routes, dst)
	} else {
		info = append(info[:delIndex], info[delIndex+1:]...)
	}
	r.Unlock()
	return true
}

func (r *RouteTable) DeleteNextRoute(next uint16) bool {
	nextList := make([]*RouteInfo, 0, 5)
	dst := make([]uint16, 0, 5)
	r.Lock()
	for u, infos := range r.routes {
		for _, info := range infos {
			if info.Next == next {
				dst = append(dst, u)
				nextList = append(nextList, info)
			}
		}
	}
	r.Unlock()
	for i, u := range dst {
		r.DeleteRoute(u, nextList[i].Next, nextList[i].Hop, nextList[i].ParentNode)
	}
	return true
}

func (r *RouteTable) DeleteRouteAll(dst uint16) {
	r.Lock()
	delete(r.routes, dst)
	r.Unlock()
}

func (r *RouteTable) RouteTableOutput() []byte {
	r.RLock()
	if len(r.routes) == 0 {
		r.RUnlock()
		return []byte("route is empty\n")
	}
	buf := bytes.NewBuffer(make([]byte, 0, 128))
	buf.WriteString("dest \tnext \thop  \tparent-node\ttime\t\n")
	for u, infos := range r.routes {
		for i := 0; i < len(infos); i++ {
			_, _ = fmt.Fprintf(buf, "%d \t%d \t%d  \t%d           \t%s    \t\n", u, infos[i].Next, infos[i].Hop, infos[i].ParentNode, time.UnixMilli(infos[i].UnixMilli).Format("2006-01-02 15:04:05"))
		}
		buf.Write([]byte{10})
	}
	r.RUnlock()
	return buf.Bytes()
}

func (r *RouteTable) GetDstRoutes(dst uint16) []*RouteInfo {
	r.RLock()
	v, _ := r.routes[dst]
	r.RUnlock()
	return v
}

type RouteInfo struct {
	UnixMilli  int64
	Next       uint16
	Hop        uint16
	ParentNode uint16
}

func searchRouteInfoHop(info []*RouteInfo, target uint16) int {
	left, right := 0, len(info)-1
	for left <= right {
		mid := left + (right-left)/2
		if info[mid].Hop > target {
			if mid == 0 || info[mid-1].Hop <= target {
				return mid
			}
			right = mid - 1
		} else {
			left = mid + 1
		}
	}
	return -1
}

func searchRouteInfoNext(info []*RouteInfo, next uint16) int {
	left, right := 0, len(info)-1
	result := -1 // 如果没有找到，则返回-1
	for left <= right {
		mid := left + (right-left)/2
		if info[mid].Next == next {
			// 找到目标值，但不一定是第一个出现
			result = mid // 暂存当前索引
			// 尝试向左移动以找到第一个出现
			if mid == 0 || info[mid-1].Next != next {
				// 已经是第一个，或者前一个不是目标值
				break
			}
			right = mid - 1 // 否则，继续在左侧查找
		} else if info[mid].Next < next {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return result
}

package iface

import "time"

type Router interface {
	// AddRoute 添加一条路由，如果已存在路由条目，则路由目的地下一跳不同则只有跳数小于时添加
	AddRoute(dst, via uint32, hop uint8, d time.Duration) (isAdd bool)
	// AddRouteWithCallback 添加一条路由,如果路由不存在则不会执行回调，直接添加路由；否则执行回调获取返回值（isAdd）为true时添加路由
	AddRouteWithCallback(dst, via uint32, hop uint8, d time.Duration, callback func(info RouteInfo) (isAdd bool)) (isAdd bool)
	// RemoveRoute 移除路由
	RemoveRoute(dst, via uint32) bool
	// RemoveRouteWithCallback 移除路由，如果目的路由不存在则不会调用回调，否则执行回调获取返回值（isDel）为true时删除路由
	RemoveRouteWithCallback(dst uint32, callback func(info RouteInfo) (isDel bool)) (isDel bool)
	// RemoveRouteWithDst 根据目的移除路由
	RemoveRouteWithDst(dst uint32)
	// RemoveRouteWithVia 根据下一跳移除目的
	RemoveRouteWithVia(via uint32) (affected int)
	// GetRoute 获取目的路由的下一跳
	GetRoute(dst uint32) (via uint32, hop uint8, exist bool)
	// GetRouteWithVia 过去下一跳可达所有目的路由
	GetRouteWithVia(via uint32) (dst []uint32)
	// RangeRoute 遍历路由表
	RangeRoute(f func(info RouteInfo))
}

type RouteInfo interface {
	Dst() uint32
	Via() uint32
	Hop() uint8
	Activation() time.Duration
	String() string
}

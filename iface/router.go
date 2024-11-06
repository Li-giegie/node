package iface

type Router interface {
	// AddRoute 添加一条路由，如果路由目的地下一跳不同则只有跳数小于时添加
	AddRoute(dst, via uint32, hop uint8) (id uint64)
	// RemoveRoute 移除路由
	RemoveRoute(dst, via uint32) bool
	// RemoveRouteWithDst 根据目的移除路由
	RemoveRouteWithDst(dst uint32)
	// RemoveRouteWithVia 根据下一跳移除目的
	RemoveRouteWithVia(via uint32) (affected int)
	// RemoveRouteWithId 根据Id移除
	RemoveRouteWithId(id uint64) bool
	// GetRoute 获取目的路由的下一跳
	GetRoute(dst uint32) (via uint32, exist bool)
	// GetRouteWithVia 过去下一跳可达所有目的路由
	GetRouteWithVia(via uint32) (dst []uint32)
	// RangeRoute 遍历路由表
	RangeRoute(f func(id uint64, dst uint32, via uint32, hop uint8))
}

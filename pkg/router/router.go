package router

import (
	"github.com/Li-giegie/node/pkg/router/impl_router"
)

// Router 是跨节点通信的路由实现接口，通常不需要手动添加删除，而是通过动态路由协议来完成的，
// 如果服务端节点需要桥接，每个服务端节点都应该有一个唯一的Id，
// 属于该服务端节点的客户端节点最好也能有一个唯一的节点Id，否则不保证正确的通信
// dst为目的，via为下一跳，hop为路由跳数，unixNano为距今的时间戳，paths 为路由的（从起点到终点）全路径
// 添加或删除一条路由时，如果该路由存在，那么unixNano参数必须大于已存在路由的unixNao，返回值bool决定是否成功，n大于0则删除n条路由
type Router interface {
	// AddRoute 添加路由
	AddRoute(dst, via uint32, hop uint8, unixNano int64, paths []*impl_router.RoutePath) bool
	// RemoveRoute 移除路由
	RemoveRoute(dst uint32, unixNano int64) bool
	// RemoveRouteWithVia 移除下一跳为 via的所有路由，n返回移除路由条目数
	RemoveRouteWithVia(via uint32, unixNano int64) (n int)
	// RemoveRouteWithPath 移除路径中包含path的路由
	RemoveRouteWithPath(path uint32, unixNano int64) (n int)
	// GetRoute 从路由表中获取dst路由
	GetRoute(dst uint32) (*impl_router.RouteEmpty, bool)
	// GetRouteVia 从路由表中dst路由的下一跳
	GetRouteVia(dst uint32) (uint32, bool)
	// RangeRoute 遍历路由表
	RangeRoute(callback func(*impl_router.RouteEmpty) bool)
	// ReroutingHandleFunc 添加重新计算路由处理方法
	ReroutingHandleFunc(f func(dst uint32) (*impl_router.RouteEmpty, bool))
	// Rerouting 重新计算目的路由下一跳，而不是从路由表中取路由
	Rerouting(dst uint32) (*impl_router.RouteEmpty, bool)
}

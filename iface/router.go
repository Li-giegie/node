package iface

import "github.com/Li-giegie/node/router"

// Router 是跨节点通信的路由实现接口，通常不需要手动添加删除，而是通过动态路由协议来完成的，
// 如果服务端节点需要桥接，每个服务端节点都应该有一个唯一的Id，
// 属于该服务端节点的客户端节点最好也能有一个唯一的节点Id，否则不保证正确的通信
// dst为目的，via为下一跳，hop为路由跳数，unixNano为距今的时间戳，paths 为路由的（从起点到终点）全路径
// 添加或删除一条路由时，如果该路由存在，那么unixNano参数必须大于已存在路由的unixNao，返回值bool决定是否成功，n大于0则删除n条路由
type Router interface {
	AddRoute(dst, via uint32, hop uint8, unixNano int64, paths []*router.RoutePath) bool
	RemoveRoute(dst uint32, unixNano int64) bool
	RemoveRouteWithVia(via uint32, unixNano int64) (n int)
	RemoveRouteWithPath(path uint32, unixNano int64) (n int)
	GetRoute(dst uint32) (*router.RouteEmpty, bool)
	GetRouteVia(dst uint32) (uint32, bool)
	RangeRoute(callback func(*router.RouteEmpty) bool)
}

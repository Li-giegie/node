package nodediscovery

import "github.com/Li-giegie/node/iface"

type NodeDiscoveryProtocol interface {
	// AddRoute 添加一条静态路由
	AddRoute(dst, via uint32, hop uint8)
	// RemoveRoute 删除一条静态路由
	RemoveRoute(dst, via uint32)
	// RemoveRouteWithDst 根据目的ID删除一条静态路由
	RemoveRouteWithDst(dst uint32)
	// RemoveRouteWithVia 根据下一跳删除一条静态路由
	RemoveRouteWithVia(via uint32)
	// GetRoute 获取路由信息
	GetRoute(dst uint32) (*RouteEmpty, bool)
	// RangeRoute 遍历互相有过通信的路由，对于没有通信过的节点，可能可达，但并没有计算路由，只有每次去往没去过的节点才会计算路由
	RangeRoute(callback func(*RouteEmpty) bool)
	// RangeNode 遍历所有节点
	RangeNode(callback func(root uint32, sub []*SubInfo))
}

type Node interface {
	Id() uint32
	GetAllConn() []iface.Conn
	GetConn(id uint32) (iface.Conn, bool)
	iface.Handler
}

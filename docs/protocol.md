# protocol 是node框架扩展


### NodeDiscoveryProtocol(节点动态路由发现)协议
### 创建协议函数签名
```go
// NewNodeDiscoveryProtocol n 节点 maxHop 最大跳数
NewNodeDiscoveryProtocol(n nodediscovery.Node) nodediscovery.NodeDiscoveryProtocol 
```
#### NodeDiscoveryProtocol协议实现方法
```go
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
```
# protocol 是node框架扩展

内部集成了，hello（连接心跳）、node-discovery（节点路由动态发现）协议，
协议的实现上在一定程度上不十分优雅，不限于接口设计、命名，
后续将会跟进。

### hello(心跳)协议
#### 创建协议函数签名
```go
// NewHelloProtocol 创建hello协议，h 参数为节点、interval 检查是否超时的间隔时间、timeout超时时间后发送心跳、timeoutClose超时多久后断开连接，该协议需要在节点启动前使用，否则可能无效
NewHelloProtocol(h iface.Handler, interval, timeout, timeoutClose time.Duration) hello.HelloProtocol
```

#### Hello协议实现方法
```go
// HelloProtocol
type HelloProtocol interface {
	// Stop 停止
	Stop()
	// ReStart 重启
	ReStart()
	// SetEventCallback 产生的事件回调，在这里可以记录日志
	SetEventCallback(callback func(action Event_Action, val interface{}))
}
```

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
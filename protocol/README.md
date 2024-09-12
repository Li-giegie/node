# protocol包实现了Node框架常用的tcp功能

内部集成了，hello（连接心跳）、node-discovery（节点路由动态发现）协议，
协议的实现上在一定程度上不十分优雅，不限于接口设计、命名，
后续将会跟进。

协议中定义的方法名与生命周期接口中方法名名相同时，需显式的在生命周期方法中调用，
返回值next为false时该协议被执行，只有为true是才能进入下一个逻辑
### hello(心跳)协议
#### 创建协议函数签名
```go
// interval time.Duration      检查超时间隔时间
// timeout time.Duration       超时时长,如果在单位时间没有读写将会发送心跳包
// timeoutClose time.Duration  超时断开，超时多久无响应连接断开
// output io.Writer            心跳包日志输出
NewHelloProtocol(interval time.Duration, timeout time.Duration, timeoutClose time.Duration, output io.Writer) HelloProtocol
```

#### Hello协议实现方法
```go
// KeepAlive 单节点保活，通常用在客户端节点
// KeepAliveMultiple 多节点保活，通常用在服务端节点
// CustomHandle 需要在连接生命周期同名函数中回调，返回值next决定当前协议是否被执行，如果执行返回false
// Stop 停止协议
type HelloProtocol interface {
	KeepAlive(c common.Conn)
	KeepAliveMultiple(conns common.Connections)
	CustomHandle(ctx common.CustomContext) (next bool)
	Stop()
}
```

### NodeDiscoveryProtocol(节点动态路由发现)协议
### 创建协议函数签名
```go
//n 接口服务端接口已经实现所有方法，但结构需要重新定义，out 输入日志
NewNodeDiscoveryProtocol(n node_discovery.DiscoveryNode, out io.Writer) NodeDiscoveryProtocol
```
#### NodeDiscoveryProtocol协议实现方法
```go
// StartTimingQueryEnableProtoNode [可选] 开启超时查询节点是否启用协议
// 在回调生命周期调用其他方法
type NodeDiscoveryProtocol interface {
	StartTimingQueryEnableProtoNode(ctx context.Context, timeout time.Duration) (err error)
	Connection(conn common.Conn)
	CustomHandle(ctx common.Context) (next bool)
	Disconnect(id uint16, err error)
}
```
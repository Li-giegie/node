# protocol包实现了Node框架常用的tcp功能

为了减少重复开发，内部集成了，authentication（节点认证）、
hello（连接心跳）、node-discovery（节点路由动态发现）协议，
协议的实现上在一定程度上不十分优雅，不限于接口设计、命名，
后续将会跟进。

### hello(心跳)协议
区分客户端和服务端认证，接口中同生命周期相同的函数需要在
node.Handler接口中自行回调

函数签名中的入参
```go
interval time.Duration      检查超时间隔时间
timeout time.Duration       超时时长,如果在单位时间没有读写将会发送心跳包
timeoutClose time.Duration  超时多久连接断开
output io.Writer            心跳包打印输出
```
```go
//服务端节点认证接口
type ServerHelloProtocol interface {
	StartServer(conns hello.Conns)//node.Server
	CustomHandle(ctx common.Context) (next bool)
	Stop()
}
//创建协议函数签名
NewServerHelloProtocol(interval time.Duration, timeout time.Duration, timeoutClose time.Duration, output io.Writer) ServerHelloProtocol 
```
//客户端节点认证接口
```go
type ClientHelloProtocol interface {
	StartClient(conn common.Conn) error //common.Conn
	CustomHandle(ctx common.Context) (next bool)
	Stop()
}
//创建协议函数签名
NewClientHelloProtocol(interval time.Duration, timeout time.Duration, timeoutClose time.Duration, output io.Writer)
```

### authentication(节点认证)协议
区分客户端和服务端认证，接口中同生命周期相同的函数需要在
node.Handler接口中自行回调

函数签名中的入参
```go
id uint16               本地ID        
key string              认证的key
timeout time.Duration   超时,在多久没有响应关闭连接

```
//客户端认证接口
```go
type ClientAuthProtocol interface {
	Init(conn net.Conn) (remoteId uint16, err error)
}
//创建协议函数签名
NewClientAuthProtocol(id uint16, key string, timeout time.Duration) ClientAuthProtocol
```

//服务端认证接口
```go
type ClientAuthProtocol interface {
	Init(conn net.Conn) (remoteId uint16, err error)
}
//创建协议函数签名
NewServerAuthProtocol(id uint16, key string, timeout time.Duration) ServerAuthProtocol
```

### node-discovery(节点动态路由发现)协议
该协议开启中服务端中
接口中同生命周期相同的函数需要在
node.Handler接口中自行回调

函数签名中的入参：node.Server接口

//服务端接口
```go
type NodeDiscoveryProtocol interface {
	StartTimingQueryEnableProtoNode(ctx context.Context, timeout time.Duration) (err error)
	Connection(conn common.Conn)
	CustomHandle(ctx common.Context) (next bool)
	Disconnect(id uint16, err error)
}
//创建协议函数签名
NewNodeDiscoveryProtocol(n node_discovery.DiscoveryNode) NodeDiscoveryProtocol
```
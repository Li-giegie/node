# protocol 
protocol是node框架的功能扩展，不是用来区分不同的业务请求，所有的业务请求都应该使用默认提供的请求方法。
```go
type Protocol interface {
    ProtocolType() uint8 // 协议的消息类型
    handler.Handler      // 协议实现方法
}

type Handler interface {
    OnAccept(conn net.Conn) (allow bool)
    OnConnect(conn conn.Conn)
    OnMessage(r responsewriter.ResponseWriter, msg *message.Message)
    OnClose(conn conn.Conn, err error)
}

```

## routerbfs
routerbfs是一个实现了节点路由功能的协议，开启该协议后节点拥有和其他服务类型节点的通信能力。

### 接口定义
```go
type Router interface {
	Protocol
	StartNodeSync(ctx context.Context, timeout time.Duration)
}
```
StartNodeSync：开启节点同步功能，开启后节点间拥有错误纠错能力。

### 概述
服务类型的节点桥接到其他服务类型的节点组成一个更大的域时，
新节点的加入和退出，会触发广播更新，收到更新的节点会计算相应的路由。
package iface

type Handler interface {
	// AddOnConnect 连接认证通过的回调, 同步调用
	AddOnConnect(OnConnectFunc)
	// AddOnMessage  收到框架内部定义的标准类型消息回调, 同步调用
	AddOnMessage(OnMessageFunc)
	// AddOnProtocolMessage 收到自定义协议消息类型的回调，协议适用于扩展功能，并不是用来区别场景的，所有场景都应该在AddOnMessage回调中实现,同步调用
	AddOnProtocolMessage(typ uint8, callback OnProtocolMessage)
	// AddOnClose 添加连接断开回调, 同步调用
	AddOnClose(OnCloseFunc)
	// AddOnRouteMessage 收到非本地节点的消息时触发，这个回调用于路由协议，不同的节点有不同的默认行为，同步调用
	//当节点是服务端节点时，如果该回调为空，则默认回复节点不存在错误
	//当节点是客户端节点时，不应该收到目的节点非当前节点的消息，该回调为空时也没有默认行为，丢弃该消息
	AddOnRouteMessage(OnRouteMessageFunc)
}

type (
	OnConnectFunc      func(conn Conn)
	OnMessageFunc      func(ctx Context)
	OnProtocolMessage  func(ctx Context)
	OnCloseFunc        func(conn Conn, err error)
	OnRouteMessageFunc func(ctx Context)
)

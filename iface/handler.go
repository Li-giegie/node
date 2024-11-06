package iface

type Handler interface {
	// AddOnClosed 添加连接断开回调, 同步回调
	AddOnClosed(callback func(conn Conn, err error))
	// AddOnCustomMessage 收到自定义类型消息回调, 同步回调
	AddOnCustomMessage(callback func(ctx Context))
	// AddOnMessage  收到框架内部定义的标准类型消息回调, 同步回调
	AddOnMessage(callback func(ctx Context))
	// AddOnConnection 连接认证通过的回调, 同步回调
	AddOnConnection(callback func(conn Conn))
	// AddOnNoRouteMessage 收到非本地节点的消息并且没有路由时触发，同步调用,
	//当节点是服务端节点时，如果该回调为空，则默认回复节点不存在错误
	//当节点是客户端节点时，不应该收到目的节点非本地节点的消息，该回调为空时也没有默认行为，丢弃该消息
	AddOnNoRouteMessage(callback func(ctx Context))
}

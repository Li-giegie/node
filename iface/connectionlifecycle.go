package iface

import "net"

// ConnectionLifecycle 连接生命周期
type ConnectionLifecycle interface {
	// OnAccept accept 行后的回到，allow是否允许接受连接，nil值默认接受
	OnAccept(conn net.Conn) (allow bool)
	// OnConnect 连接通过基础认证正式建立后的回调
	OnConnect(conn Conn)
	// OnMessage 收到消息后的回调
	OnMessage(ctx Context)
	// OnClose 连接关闭后的回调
	OnClose(conn Conn, err error)
}

type OnAcceptFunc func(conn net.Conn) (allow bool)
type OnConnectFunc func(conn Conn)
type OnMessageFunc func(ctx Context)
type OnCloseFunc func(conn Conn, err error)

type ConnectionLifecycleCallback interface {
	// OnAccept accept 行后的回到，allow是否允许接受连接，nil值默认接受
	OnAccept(callback OnAcceptFunc)
	// OnConnect 连接通过基础认证正式建立后的回调
	OnConnect(callback OnConnectFunc)
	// OnMessage 收到消息后的回调
	OnMessage(callback OnMessageFunc)
	// OnClose 连接关闭后的回调
	OnClose(callback OnCloseFunc)
}

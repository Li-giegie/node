package iface

import (
	"net"
	"time"
)

type Server interface {
	// Serve 开启服务
	Serve(l net.Listener) error
	//Bridge 从当前节点桥接一个节点,组成一个更大的域，如果要完整启用该功能则需要开启节点动态发现协议
	Bridge(conn net.Conn, remoteId uint32, remoteAuthKey []byte, timeout time.Duration) (err error)
	// GetConn 获取连接
	GetConn(id uint32) (Conn, bool)
	// GetAllConn 获取所有连接
	GetAllConn() []Conn
	// Id 当前节点ID
	Id() uint32
	// Close 关闭服务
	Close()
	// AddOnAcceptConnect 接受连接回调，返回值isClose为true则关闭连接，同步调用回调直到完成本次回调才能接受下一个连接，耗时逻辑请开启协程
	AddOnAcceptConnect(OnAcceptConnectFunc)
	Handler
}

// OnAcceptConnectFunc 接受连接回调，返回值isClose为true则关闭连接
type OnAcceptConnectFunc func(conn net.Conn) (isClose bool)

type AcceptConnection interface {
	AddOnAcceptConnect(OnAcceptConnectFunc)
}
package iface

import (
	"context"
	"net"
	"time"
)

type Server interface {
	// Serve 开启服务
	Serve() error
	// Bridge 在当前域绑定另一个域，组成一个大的域，如果要完整启用该功能则需要开启节点动态发现协议
	Bridge(conn net.Conn, remoteAuthKey []byte, timeout time.Duration) (rid uint32, err error)
	// Request 请求并得到响应
	Request(ctx context.Context, dst uint32, data []byte) ([]byte, error)
	// WriteTo 发送消息
	WriteTo(dst uint32, data []byte) (int, error)
	// GetConn 获取连接
	GetConn(id uint32) (Conn, bool)
	// GetAllConn 获取所有连接
	GetAllConn() []Conn
	// GetAllId 获取所有连接ID
	GetAllId() []uint32
	// Id 当前节点ID
	Id() uint32
	// Close 关闭服务
	Close() error
	Router
	Handler
}

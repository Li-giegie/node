package iface

import (
	"context"
	"crypto/tls"
	"github.com/Li-giegie/node/message"
	"net"
)

type Server interface {
	// Serve 开启服务
	Serve(l net.Listener) error
	// ListenAndServe 侦听并开启服务,address 支持url格式例如 tcp://127.0.0.1:5555 = 127.0.0.1:5555，缺省协议默认tcp
	ListenAndServe(address string, conf ...*tls.Config) (err error)
	//Bridge 从当前节点桥接一个节点,组成一个更大的域，如果要完整启用该功能则需要开启节点动态发现协议
	Bridge(conn net.Conn, remoteId uint32, remoteAuthKey []byte) (err error)
	// GetConn 获取连接
	GetConn(id uint32) (Conn, bool)
	// GetAllConn 获取所有连接
	GetAllConn() []Conn
	// Id 当前节点ID
	Id() uint32
	// Close 关闭服务
	Close()
	GetRouter() Router
	ConnectionLifecycleCallback
	RequestTo(ctx context.Context, dst uint32, data []byte) (response []byte, stateCode int16, err error)
	RequestTypeTo(ctx context.Context, typ uint8, dst uint32, data []byte) (response []byte, stateCode int16, err error)
	RequestMessage(ctx context.Context, msg *message.Message) (response []byte, stateCode int16, err error)
	SendTo(dst uint32, data []byte) error
	SendTypeTo(typ uint8, dst uint32, data []byte) error
	SendMessage(m *message.Message) error
}

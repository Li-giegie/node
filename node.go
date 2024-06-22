package node

import (
	"github.com/Li-giegie/node/common"
	"net"
)

type Node interface {
	// Connection 连接被建立时触发回调
	Connection(conn net.Conn) (remoteId uint16, err error)
	// Handle 接收到标准类型消息时触发回调
	Handle(ctx *common.Context)
	// ErrHandle 发送失败触发的回调
	ErrHandle(msg *common.Message)
	// DropHandle 接收到超时消息时触发回调
	DropHandle(msg *common.Message)
	// CustomHandle 接收到自定义类型消息时触发回调
	CustomHandle(ctx *common.Context)
	// Disconnect 连接断开触发回调
	Disconnect(id uint16, err error)
}

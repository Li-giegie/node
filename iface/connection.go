package iface

import (
	"context"
	"github.com/Li-giegie/node/message"
)

type Conn interface {
	// Request 请求并得到响应
	Request(ctx context.Context, data []byte) ([]byte, error)
	// Forward 转发请求，并得到响应
	Forward(ctx context.Context, destId uint32, data []byte) ([]byte, error)
	// Write 发送
	Write(data []byte) (n int, err error)
	// WriteTo 发送到目的节点
	WriteTo(dst uint32, data []byte) (n int, err error)
	// WriteMsg 发送一个自定义构建的消息
	WriteMsg(m *message.Message) (n int, err error)
	// Close 关闭连接
	Close() error
	// LocalId 本地ID
	LocalId() uint32
	// RemoteId 对端Id
	RemoteId() uint32
	// Activate 激活时间
	Activate() int64
	// NodeType 节点类型
	NodeType() uint8
	// IsClosed 是否关闭
	IsClosed() bool
}

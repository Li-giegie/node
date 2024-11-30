package iface

import (
	"context"
	"github.com/Li-giegie/node/message"
	"net"
	"time"
)

type Conn interface {
	// Request 发起请求到服务端
	Request(ctx context.Context, data []byte) ([]byte, error)
	// RequestTo 发起请求到目的节点
	RequestTo(ctx context.Context, dst uint32, data []byte) ([]byte, error)
	// RequestType 发起请求到服务端，并设置type，type字段通常用于协议
	RequestType(ctx context.Context, typ uint8, data []byte) ([]byte, error)
	// RequestTypeTo 发起请求到目的节点，并设置type，type字段通常用于协议
	RequestTypeTo(ctx context.Context, typ uint8, dst uint32, data []byte) ([]byte, error)
	// RequestMessage 发起请求，msg请使用CreateMessage创建一个完整的msg或者使用CreateMessageId创建，保证Id字段唯一性
	RequestMessage(ctx context.Context, msg *message.Message) ([]byte, error)
	// Write 发起到服务端
	Write(data []byte) (n int, err error)
	// WriteTo 发起到目的节点
	WriteTo(dst uint32, data []byte) (n int, err error)
	// WriteMessage 发送一个自定义构建的消息
	WriteMessage(m *message.Message) (n int, err error)
	// Close 关闭连接
	Close() error
	// LocalId 本地ID
	LocalId() uint32
	// RemoteId 对端Id
	RemoteId() uint32
	// Activate 激活时间
	Activate() time.Duration
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	// CreateMessage 创建一个唯一消息Id的消息，hop为0
	CreateMessage(typ uint8, src uint32, dst uint32, data []byte) *message.Message
	// CreateMessageId 创建一个唯一的消息Id
	CreateMessageId() uint32
}

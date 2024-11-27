package iface

import (
	"context"
	"github.com/Li-giegie/node/message"
	"net"
	"time"
)

type Conn interface {
	// Request 从当前节点请求并得到响应
	Request(ctx context.Context, data []byte) ([]byte, error)
	// Forward 从当前节点转发请求，并得到响应
	Forward(ctx context.Context, destId uint32, data []byte) ([]byte, error)
	// Write 从当前节点发送
	Write(data []byte) (n int, err error)
	// WriteTo 从当前节点发送到目的节点
	WriteTo(dst uint32, data []byte) (n int, err error)
	// WriteMsg 发送一个自定义构建的消息，例如修改源Id（SrcId）、修改消息Type时使用
	WriteMsg(m *message.Message) (n int, err error)
	// Close 关闭连接
	Close() error
	// LocalId 本地ID
	LocalId() uint32
	// RemoteId 对端Id
	RemoteId() uint32
	// Activate 激活时间
	Activate() time.Duration
	// NodeType 节点类型
	NodeType() uint8
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

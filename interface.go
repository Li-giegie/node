package node

import (
	"context"
	"github.com/Li-giegie/node/common"
)

type Conn interface {
	Request(ctx context.Context, data []byte) ([]byte, error)
	AsyncRequest(ctx context.Context, data []byte, callback func(res []byte, err error))
	Forward(ctx context.Context, destId uint32, data []byte) ([]byte, error)
	AsyncForward(ctx context.Context, destId uint32, data []byte, callback func(res []byte, err error))
	Write(data []byte) (n int, err error)
	WriteTo(dst uint32, data []byte) (n int, err error)
	// WriteMsg 构建一条自定义类型的消息发送
	WriteMsg(m *common.Message) (n int, err error)
	Close() error
	State() uint8
	LocalId() uint32
	RemoteId() uint32
	Activate() int64
}

/*
Handler
Connection 连接第一次建立成功回调
Handle 接到标准类型消息会被触发，如果该回调阻塞将阻塞当前节点整个生命周期回调（在同步调用模式中如果在这个回调中发起请求需要另外开启协程否则会陷入阻塞，无法接收到消息），框架并没有集成协程池，第三方框架众多，合理选择
ErrHandle 当收到：超过限制长度的消息0xffffff、校验和错误、超时、服务节点返回的消息但没有接收的消息都会在这里触发回调
CustomHandle 自定义消息类型处理，框架内部默认集成了多种消息类型，当需要一些特定的功能时可以自定义消息类型，例如心跳消息，只需把消息类型声明成框架内部不存在的类型，框架看到不认识的消息就会回调当前函数
Disconnect 连接断开会被触发
*/
type Handler interface {
	// Connection 连接第一次建立成功回调
	Connection(conn common.Conn)
	// Handle 接收到标准类型消息时触发回调
	Handle(ctx common.Context)
	// ErrHandle 发送失败触发的回调
	ErrHandle(ctx common.ErrContext, err error)
	// CustomHandle 接收到自定义类型消息时触发回调
	CustomHandle(ctx common.CustomContext)
	// Disconnect 连接断开触发回调
	Disconnect(id uint32, err error)
}

package node

import (
	"context"
	"github.com/Li-giegie/node/common"
)

type Conn interface {
	Request(ctx context.Context, data []byte) ([]byte, error)
	Forward(ctx context.Context, destId uint16, data []byte) ([]byte, error)
	Write(data []byte) (n int, err error)
	WriteTo(dst uint16, data []byte) (n int, err error)
	Close() error
	State() uint8
	WriteMsg(m *common.Message) (err error)
	LocalId() uint16
	RemoteId() uint16
	Activate() int64
}

/*
Connection 同步调用，连接第一次建立成功回调
Handle 默认同步调用：异同步取决于你，如果该回调阻塞将影响当前连接整个生命周期（对于一些不消耗时间的任务，重新开启一个goroutine执行未必最优），框架并没有集成协程池，第三方框架众多，一时拿不定主意，索性把问题抛给你
ErrHandle 默认同步调用：异同步取决于你，当发送消息失败时会被触发
CustomHandle 默认同步调用：异同步取决于你，自定义消息类型处理，框架内部集成了多种消息类型，当需要一些特定的功能时可以自定义消息类型，例如心跳消息，只需把消息类型声明成框架内部不存在的类型，框架看到不认识的消息就会回调当前函数
Disconnect 同步调用：连接断开会被触发
*/
type Handler interface {
	// Connection 同步调用，连接第一次建立成功回调
	Connection(conn common.Conn)
	// Handle 接收到标准类型消息时触发回调
	Handle(ctx common.Context)
	// ErrHandle 发送失败触发的回调
	ErrHandle(ctx common.ErrContext, err error)
	// CustomHandle 接收到自定义类型消息时触发回调
	CustomHandle(ctx common.CustomContext)
	// Disconnect 连接断开触发回调
	Disconnect(id uint16, err error)
}

package node

import (
	"github.com/Li-giegie/node/common"
	"net"
)

/*
Node 一个连接的生命周期

# Connection 同步调用：该方法在连接建立时被调用，返回对端ID、error，如果err返回值非空，连接关闭，此步骤用于自定义认证，在服务端返回空，且ID唯一（连接列表不存在）连接会被建立，反之服务端会关闭连接并返回连接已存在错误

# Handle 默认同步调用：异同步取决于你，如果该回调阻塞将影响当前连接整个生命周期（对于一些不消耗时间的任务，重新开启一个goroutine执行未必最优），框架并没有集成协程池，第三方框架众多，一时拿不定主意，索性把问题抛给你

# ErrHandle 默认同步调用：异同步取决于你，当发送消息失败时会被触发

# DropHandle 默认同步调用：异同步取决于你，目前仅会在接受消息超时时会被回调

# CustomHandle 默认同步调用：异同步取决于你，自定义消息类型处理，框架内部集成了多种消息类型，当需要一些特定的功能时可以自定义消息类型，例如心跳消息，只需把消息类型声明成框架内部不存在的类型，框架看到不认识的消息就会回调当前函数

Disconnect 同步调用：连接断开会被触发
*/
type Node interface {
	// Connection 连接被建立时触发回调
	Connection(conn net.Conn) (remoteId uint16, err error)
	// Handle 接收到标准类型消息时触发回调
	Handle(ctx common.Context)
	// ErrHandle 发送失败触发的回调
	ErrHandle(msg *common.Message)
	// DropHandle 接收到超时消息时触发回调
	DropHandle(msg *common.Message)
	// CustomHandle 接收到自定义类型消息时触发回调
	CustomHandle(ctx common.Context)
	// Disconnect 连接断开触发回调
	Disconnect(id uint16, err error)
}

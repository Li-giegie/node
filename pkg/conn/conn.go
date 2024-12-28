package conn

import (
	"context"
	"github.com/Li-giegie/node/pkg/message"
	"net"
	"time"
)

// Conn 接口实现了对连接的发送、请求、关闭，和一些属性查询方法
// Type 字段用于实现某些缺失功能协议，而不是用于区别不同场景，不可滥用，所有业务场景都能在Data字段内封装实现
// Request、Send系列方法实现了核心功能，发出的消息Type属性字段均为 MsgType_Default，他们在逻辑上不同，还有Request有一个
// 收发队列而Send没有，在OnMessage等回调函数哪里看到的就是一个消息可以响应，也可以不响应，具体根据你的场景
// 如果使用了RequestType、RequestTypeTo、RequestMessage等变更Type的方法时，响应方法Response返回的类型必须是MsgType_Reply（默认就是），
// 否则接收端收到消息后无法进入响应队列从而得不到响应结果，会进入OnProtocolMessage回调那么就不能认为这是一次请求，因为和发送没有区别。
// 如果需要构造message发送时需要注意id、type、hop字段，type上文有提到，id字段为消息的唯一标识，当发起请求时，id必须保证唯一，
// 当要使用RequestMessage方法时就需要给一个唯一id，其他的方法内部已经维护好了自增id，CreateMessage可以创建一个维护好的id消息，CreateMessageId则可以创建一个唯一Id，
// hop 字段为消息的跳数，主要用在路由协议中，初始值为0，每经过一个节点都会加1，如果不是转发，初始值都应该是0,
type Conn interface {
	// Request 发起请求到直连的服务端
	Request(ctx context.Context, data []byte) (response []byte, stateCode int16, err error)
	// RequestTo 发起请求到目的节点，当目的节点不是直连节点时使用此方法
	RequestTo(ctx context.Context, dst uint32, data []byte) (response []byte, stateCode int16, err error)
	// RequestType 发起请求到服务端，并指定type，type字段通常用于协议
	RequestType(ctx context.Context, typ uint8, data []byte) (response []byte, stateCode int16, err error)
	// RequestTypeTo 发起请求到目的节点，并指定type，type字段通常用于协议,当目的节点不是直连节点时使用此方法
	RequestTypeTo(ctx context.Context, typ uint8, dst uint32, data []byte) (response []byte, stateCode int16, err error)
	// RequestMessage 发起请求，并自主构建完整的消息，注意：msg请使用CreateMessage创建一个完整的msg或者使用CreateMessageId创建，保证Id字段唯一性
	RequestMessage(ctx context.Context, msg *message.Message) (response []byte, stateCode int16, err error)
	// Send 发送数据到服务端
	Send(data []byte) error
	// SendTo 发送数据到目的节点，当目的节点不是直连节点时使用此方法
	SendTo(dst uint32, data []byte) error
	// SendType 发送数据到服务端，并指定type，type字段通常用于协议
	SendType(typ uint8, data []byte) error
	// SendTypeTo 发送数据到目的节点，并设置type，type字段通常用于协议，当目的节点不是直连节点时使用此方法
	SendTypeTo(typ uint8, dst uint32, data []byte) error
	// SendMessage 发送一个自主构建的消息
	SendMessage(m *message.Message) error
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

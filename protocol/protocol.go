package protocol

import (
	"github.com/Li-giegie/node/message"
	"github.com/Li-giegie/node/protocol/hello"
	"github.com/Li-giegie/node/protocol/routerbfs"
	"time"
)

var defaultMsgType = message.MsgType_Undefined

func CreateProtocolMsgType() uint8 {
	defaultMsgType++
	return defaultMsgType
}

var (
	protoMsgType_Hello         = CreateProtocolMsgType()
	protoMsgType_NodeDiscovery = CreateProtocolMsgType()
)

// NewHelloProtocol 创建hello协议，h 参数为节点、interval 检查是否超时的间隔时间、timeout超时时间后发送心跳、timeoutClose超时多久后断开连接，该协议需要在节点启动前使用，否则可能无效
func NewHelloProtocol(interval, timeout, timeoutClose time.Duration) hello.Protocol {
	return hello.NewHelloProtocol(protoMsgType_Hello, interval, timeout, timeoutClose)
}

// NewRouterBFSProtocol BFS 路由协议 n 节点 maxHop 最大跳数
func NewRouterBFSProtocol(node routerbfs.Node) routerbfs.Protocol {
	return routerbfs.NewRouterBFS(protoMsgType_NodeDiscovery, node, 32, time.Second*15, time.Second*15)
}

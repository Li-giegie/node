package protocol

import (
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	"github.com/Li-giegie/node/protocol/hello"
	"github.com/Li-giegie/node/protocol/nodediscovery"
	"time"
)

var defaultMsgType = message.Null

func GetMsgType() uint8 {
	defaultMsgType++
	return defaultMsgType
}

var (
	protoMsgType_Hello         = GetMsgType()
	protoMsgType_NodeDiscovery = GetMsgType()
)

// NewHelloProtocol 创建hello协议，h 参数为节点、interval 检查是否超时的间隔时间、timeout超时时间后发送心跳、timeoutClose超时多久后断开连接，该协议需要在节点启动前使用，否则可能无效
func NewHelloProtocol(h iface.Handler, interval, timeout, timeoutClose time.Duration) hello.HelloProtocol {
	return hello.NewHelloProtocol(protoMsgType_Hello, h, interval, timeout, timeoutClose)
}

// NewNodeDiscoveryProtocol n 节点 maxHop 最大跳数
func NewNodeDiscoveryProtocol(n nodediscovery.Node) nodediscovery.NodeDiscoveryProtocol {
	return nodediscovery.NewNodeDiscovery(protoMsgType_NodeDiscovery, n, 32, time.Second*15)
}

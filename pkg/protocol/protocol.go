package protocol

import (
	"github.com/Li-giegie/node/pkg/handler"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/protocol/routerbfs"
	"time"
)

type Protocol interface {
	ProtocolType() uint8
	handler.Handler
}

var defaultMsgType = message.MsgType_Undefined

func CreateProtocolMsgType() uint8 {
	defaultMsgType++
	return defaultMsgType
}

var (
	protoMsgType_NodeDiscovery = CreateProtocolMsgType()
)

// NewRouterBFSProtocol BFS 路由协议 n 节点 maxHop 最大跳数
func NewRouterBFSProtocol(node routerbfs.Node) Protocol {
	return routerbfs.NewRouterBFS(protoMsgType_NodeDiscovery, node, 32, time.Second*15, time.Second*15)
}

package protocol

import (
	"context"
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
	ProtocolType_RouteBFS = CreateProtocolMsgType()
)

type Router interface {
	Protocol
	StartNodeSync(ctx context.Context, timeout time.Duration)
}

// NewRouterBFSProtocol BFS 路由协议
func NewRouterBFSProtocol(node routerbfs.Node) Router {
	return routerbfs.NewRouterBFS(ProtocolType_RouteBFS, node)
}

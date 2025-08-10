package protocol

import (
	"context"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/protocol/routerbfs"
	"github.com/Li-giegie/node/pkg/reply"
	"time"
)

var (
	defaultMsgType        = message.MsgType_Undefined
	ProtocolType_RouteBFS = CreateProtocolMsgType()
)

func CreateProtocolMsgType() uint8 {
	defaultMsgType++
	return defaultMsgType
}

type Router interface {
	StartNodeSync(ctx context.Context, timeout time.Duration)
	OnMessage(r *reply.Reply, msg *message.Message) bool
	OnConnect(c *conn.Conn) bool
	OnClose(c *conn.Conn, err error) bool
}

// NewRouterBFSProtocol BFS 路由协议
func NewRouterBFSProtocol(node node.Server) Router {
	return routerbfs.NewRouterBFS(ProtocolType_RouteBFS, node)
}

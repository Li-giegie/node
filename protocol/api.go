package protocol

import (
	"context"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/protocol/hello"
	node_discovery "github.com/Li-giegie/node/protocol/node-discovery"
	"io"
	"time"
)

var (
	protoMsgType_Hello_Send    = GetMsgType()
	protoMsgType_Hello_Reply   = GetMsgType()
	protoMsgType_NodeDiscovery = GetMsgType()
)

type NodeDiscoveryProtocol interface {
	StartTimingQueryEnableProtoNode(ctx context.Context, timeout time.Duration) (err error)
	Connection(conn common.Conn)
	CustomHandle(ctx common.CustomContext) (next bool)
	Disconnect(id uint32, err error)
}

func NewNodeDiscoveryProtocol(n node_discovery.DiscoveryNode, out io.Writer) NodeDiscoveryProtocol {
	return node_discovery.NewNodeDiscoveryProtocol(n, protoMsgType_NodeDiscovery, out)
}

type HelloProtocol interface {
	KeepAlive(c common.Conn)
	KeepAliveMultiple(conns common.Connections)
	CustomHandle(ctx common.CustomContext) (next bool)
	Stop()
}

func NewHelloProtocol(interval time.Duration, timeout time.Duration, timeoutClose time.Duration, output io.Writer) HelloProtocol {
	return hello.NewHelloProtocol(protoMsgType_Hello_Send, protoMsgType_Hello_Reply, interval, timeout, timeoutClose, output)
}

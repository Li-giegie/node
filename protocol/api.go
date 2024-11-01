package protocol

import (
	"context"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/protocol/hello"
	node_discovery "github.com/Li-giegie/node/protocol/nodediscovery"
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
	Connection(conn node.Conn)
	CustomHandle(ctx node.CustomContext) (next bool)
	Disconnect(id uint32, err error)
}

func NewNodeDiscoveryProtocol(id uint32, conns node_discovery.Conns, router node_discovery.Router, out io.Writer) NodeDiscoveryProtocol {
	return node_discovery.NewNodeDiscoveryProtocol(id, conns, router, protoMsgType_NodeDiscovery, out)
}

type HelloProtocol interface {
	KeepAlive(c node.Conn)
	KeepAliveMultiple(conns hello.Conns)
	OnCustomMessage(ctx node.CustomContext) (next bool)
	Stop()
}

func NewHelloProtocol(interval time.Duration, timeout time.Duration, timeoutClose time.Duration, output io.Writer) HelloProtocol {
	return hello.NewHelloProtocol(protoMsgType_Hello_Send, protoMsgType_Hello_Reply, interval, timeout, timeoutClose, output)
}

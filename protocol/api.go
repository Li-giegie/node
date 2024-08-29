package protocol

import (
	"context"
	"fmt"
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
	// StartTimingQueryEnableProtoNode 开启超时广播查询节点信息，可以不开启
	StartTimingQueryEnableProtoNode(ctx context.Context, timeout time.Duration) (err error)
	// Connection 需在生命周期内显示调用
	Connection(conn common.Conn)
	// CustomHandle 需在生命周期内显示调用
	CustomHandle(ctx common.CustomContext) (next bool)
	// Disconnect 需在生命周期内显示调用
	Disconnect(id uint16, err error)
}

func NewNodeDiscoveryProtocol(n node_discovery.DiscoveryNode, out io.Writer) NodeDiscoveryProtocol {
	return node_discovery.NewNodeDiscoveryProtocol(n, protoMsgType_NodeDiscovery, out)
}

type HelloProtocol interface {
	// KeepAlive 需显示调用 维持一个连接
	KeepAlive(c common.Conn)
	// KeepAliveMultiple 需显示调用 维持多个连接，这通常用作服务节点
	KeepAliveMultiple(conns common.Connections)
	// CustomHandle 需在生命周期内显示调用
	CustomHandle(ctx common.CustomContext) (next bool)
	Stop()
}

func NewHelloProtocol(interval time.Duration, timeout time.Duration, timeoutClose time.Duration, output io.Writer) HelloProtocol {
	fmt.Println(protoMsgType_Hello_Send, protoMsgType_Hello_Reply)
	return hello.NewHelloProtocol(protoMsgType_Hello_Send, protoMsgType_Hello_Reply, interval, timeout, timeoutClose, output)
}

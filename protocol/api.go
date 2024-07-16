package protocol

import (
	"context"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/protocol/authentication"
	"github.com/Li-giegie/node/protocol/hello"
	node_discovery "github.com/Li-giegie/node/protocol/node-discovery"
	"io"
	"net"
	"time"
)

type NodeDiscoveryProtocol interface {
	StartTimingQueryEnableProtoNode(ctx context.Context, timeout time.Duration) (err error)
	Connection(conn common.Conn)
	CustomHandle(ctx common.Context) (next bool)
	Disconnect(id uint16, err error)
}

func NewNodeDiscoveryProtocol(n node_discovery.DiscoveryNode) NodeDiscoveryProtocol {
	return node_discovery.NewNodeDiscoveryProtocol(n, GetMsgType())
}

type ClientAuthProtocol interface {
	Init(conn net.Conn) (remoteId uint16, err error)
}

func NewClientAuthProtocol(id uint16, key string, timeout time.Duration) ClientAuthProtocol {
	return authentication.NewClientAuthProtocol(id, key, timeout)
}

type ServerAuthProtocol interface {
	Init(conn net.Conn) (remoteId uint16, err error)
}

func NewServerAuthProtocol(id uint16, key string, timeout time.Duration) ServerAuthProtocol {
	return authentication.NewServerAuthProtocol(id, key, timeout)
}

type ServerHelloProtocol interface {
	StartServer(conns hello.Conns)
	CustomHandle(ctx common.Context) (next bool)
	Stop()
}

func NewServerHelloProtocol(interval time.Duration, timeout time.Duration, timeoutClose time.Duration, output io.Writer) ServerHelloProtocol {
	return hello.NewHelloProtocol(GetMsgType(), GetMsgType(), interval, timeout, timeoutClose, output)
}

type ClientHelloProtocol interface {
	StartClient(conn common.Conn) error
	CustomHandle(ctx common.Context) (next bool)
	Stop()
}

func NewClientHelloProtocol(interval time.Duration, timeout time.Duration, timeoutClose time.Duration, output io.Writer) ClientHelloProtocol {
	return hello.NewHelloProtocol(GetMsgType(), GetMsgType(), interval, timeout, timeoutClose, output)
}

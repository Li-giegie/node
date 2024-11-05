package protocol

import (
	"context"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	"github.com/Li-giegie/node/protocol/hello"
	"github.com/Li-giegie/node/protocol/nodediscovery"
	"io"
	"time"
)

var defaultMsgType = message.Null

func GetMsgType() uint8 {
	defaultMsgType++
	return defaultMsgType
}

var (
	protoMsgType_Hello_Send    = GetMsgType()
	protoMsgType_Hello_Reply   = GetMsgType()
	protoMsgType_NodeDiscovery = GetMsgType()
)

type HelloProtocol interface {
	KeepAlive(c iface.Conn)
	KeepAliveMultiple(conns hello.Conns)
	Stop()
}

func NewHelloProtocol(h iface.Handler, interval time.Duration, timeout time.Duration, timeoutClose time.Duration, output io.Writer) HelloProtocol {
	p := hello.HelloProtocol{
		HelloProtocolMsgType_Send:  protoMsgType_Hello_Send,
		HelloProtocolMsgType_Reply: protoMsgType_Hello_Reply,
		Timeout:                    timeout,
		TimeoutClose:               timeoutClose,
		Output:                     output,
		Ticker:                     time.NewTicker(interval),
	}
	h.AddOnCustomMessage(p.OnCustomMessage)
	return &p
}

func StartHelloProtocol(ctx context.Context, conn iface.Conn, h iface.Handler, interval time.Duration, timeout time.Duration, timeoutClose time.Duration, output io.Writer) {
	p := NewHelloProtocol(h, interval, timeout, timeoutClose, output)
	go p.KeepAlive(conn)
	go func() {
		if ctx != nil {
			<-ctx.Done()
			p.Stop()
		}
	}()
}

func StartMultipleNodeHelloProtocol(ctx context.Context, conns hello.Conns, h iface.Handler, interval time.Duration, timeout time.Duration, timeoutClose time.Duration, output io.Writer) {
	p := NewHelloProtocol(h, interval, timeout, timeoutClose, output)
	go p.KeepAliveMultiple(conns)
	go func() {
		if ctx != nil {
			<-ctx.Done()
			p.Stop()
		}
	}()
}

func StartDiscoveryProtocol(maxHop uint8, h iface.Handler, n nodediscovery.Node) {
	p := nodediscovery.NodeDiscovery{
		ProtoMsgType: protoMsgType_NodeDiscovery,
		Node:         n,
		MaxHop:       maxHop,
	}
	h.AddOnConnection(p.OnConnection)
	h.AddOnCustomMessage(p.OnCustomMessage)
	h.AddOnClosed(p.OnClose)
}

package hello

import (
	"fmt"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	"io"
	"time"
)

type HelloProtocol struct {
	HelloProtocolMsgType_Send  uint8
	HelloProtocolMsgType_Reply uint8
	Timeout, TimeoutClose      time.Duration
	Output                     io.Writer
	*time.Ticker
}

// KeepAlive 维持一个连接
func (h *HelloProtocol) KeepAlive(c iface.Conn) {
	for _ = range h.Ticker.C {
		if c.IsClosed() {
			h.Ticker.Stop()
			return
		}
		if h.handle(c) {
			h.Ticker.Stop()
			return
		}
	}
}

type Conns interface {
	GetAllConn() []iface.Conn
}

// KeepAliveMultiple 维持多个连接，这通常用作服务节点
func (h *HelloProtocol) KeepAliveMultiple(conns Conns) {
	for _ = range h.Ticker.C {
		for _, conn := range conns.GetAllConn() {
			h.handle(conn)
		}
	}
}

func (h *HelloProtocol) WriteOutput(data []byte) {
	if h.Output != nil {
		h.Output.Write(data)
	}
}

func (h *HelloProtocol) WriteOutputLn(arg ...any) {
	h.WriteOutput([]byte(fmt.Sprintln(arg...)))
}

func (h *HelloProtocol) handle(c iface.Conn) (closed bool) {
	msg := new(message.Message)
	msg.Type = h.HelloProtocolMsgType_Send
	msg.SrcId = c.LocalId()
	msg.DestId = c.RemoteId()
	duration := time.Now().UnixMilli() - c.Activate()
	if duration > h.TimeoutClose.Milliseconds() {
		_ = c.Close()
		h.WriteOutputLn("conn timeout close")
		return true
	}
	if duration > h.Timeout.Milliseconds() {
		if _, err := c.WriteMsg(msg); err != nil {
			_ = c.Close()
			h.WriteOutputLn("err: send hello ASK pack fail destId", c.RemoteId(), err)
			return true
		}
		h.WriteOutputLn("send hello ASK pack destId", c.RemoteId())
	}
	return false
}

func (h *HelloProtocol) OnCustomMessage(ctx iface.Context) {
	switch ctx.Type() {
	case h.HelloProtocolMsgType_Send:
		h.WriteOutputLn("received hello ASK pack srcId", ctx.SrcId())
		if err := ctx.CustomReply(h.HelloProtocolMsgType_Reply, nil); err != nil {
			h.WriteOutputLn("err: reply hello ACK pack srcId", ctx.SrcId(), "err", err)
		}
		ctx.Stop()
	case h.HelloProtocolMsgType_Reply:
		h.WriteOutputLn("received hello ACK pack srcId", ctx.SrcId())
		ctx.Stop()
	}
}

func (h *HelloProtocol) Stop() {
	h.Ticker.Stop()
}

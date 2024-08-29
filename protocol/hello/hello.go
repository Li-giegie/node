package hello

import (
	"github.com/Li-giegie/node/common"
	"io"
	"log"
	"time"
)

type HelloProtocol struct {
	HelloProtocolMsgType_Send       uint8
	HelloProtocolMsgType_Reply      uint8
	Interval, Timeout, TimeoutClose time.Duration
	*log.Logger
	*time.Ticker
}

func NewHelloProtocol(msgTypeSend, msgTypeReply uint8, interval, timeout, timeoutClose time.Duration, output io.Writer) *HelloProtocol {
	if output == nil {
		output = io.Discard
	}
	return &HelloProtocol{
		HelloProtocolMsgType_Send:  msgTypeSend,
		HelloProtocolMsgType_Reply: msgTypeReply,
		Interval:                   interval,
		Timeout:                    timeout,
		TimeoutClose:               timeoutClose,
		Logger:                     log.New(output, "[HelloProtocol] ", log.Ldate|log.Ltime|log.Lmsgprefix),
	}
}

// KeepAlive 维持一个连接
func (h *HelloProtocol) KeepAlive(c common.Conn) {
	// 连续三次状态错误、或者发送失败、超时关闭将会结束
	h.Ticker = time.NewTicker(h.Interval)
	i := 0
	for _ = range h.Ticker.C {
		if i > 2 {
			h.Ticker.Stop()
			return
		}
		if c.State() != 1 {
			i++
			continue
		}
		i = 0
		if h.handle(c) {
			h.Ticker.Stop()
			return
		}
	}
}

// KeepAliveMultiple 维持多个连接，这通常用作服务节点
func (h *HelloProtocol) KeepAliveMultiple(conns common.Connections) {
	h.Ticker = time.NewTicker(h.Interval)
	defer h.Ticker.Stop()
	for _ = range h.Ticker.C {
		for _, conn := range conns.GetConns() {
			h.handle(conn)
		}
	}
}

func (h *HelloProtocol) handle(c common.Conn) (exit bool) {
	msg := new(common.Message)
	msg.Type = h.HelloProtocolMsgType_Send
	msg.SrcId = c.LocalId()
	msg.DestId = c.RemoteId()
	duration := time.Now().UnixMilli() - c.Activate()
	if duration > h.TimeoutClose.Milliseconds() {
		_ = c.Close()
		h.Logger.Println("conn timeout close")
		return true
	}
	if duration > h.Timeout.Milliseconds() {
		if err := c.WriteMsg(msg); err != nil {
			_ = c.Close()
			h.Logger.Println("err: send hello ASK pack fail destId", c.RemoteId(), err)
			return true
		}
		h.Logger.Println("send hello ASK pack destId", c.RemoteId())
	}
	return false
}

func (h *HelloProtocol) CustomHandle(ctx common.CustomContext) (next bool) {
	switch ctx.Type() {
	case h.HelloProtocolMsgType_Send:
		h.Logger.Println("receive hello ASK pack srcId", ctx.SrcId())
		if err := ctx.CustomReply(h.HelloProtocolMsgType_Reply, nil); err != nil {
			h.Logger.Println("err: reply hello ACK pack srcId", ctx.SrcId(), "err", err)
		}
		return false
	case h.HelloProtocolMsgType_Reply:
		h.Logger.Println("receive hello ACK pack srcId", ctx.SrcId())
		return false
	default:
		return true
	}
}

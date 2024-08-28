package hello

import (
	"github.com/Li-giegie/node/common"
	"io"
	"log"
	"time"
)

type Conns interface {
	GetConns() []common.Conn
}

type HelloProtocol struct {
	running                         bool
	HelloProtocolMsgType_Send       uint8
	HelloProtocolMsgType_Reply      uint8
	Interval, Timeout, TimeoutClose time.Duration
	*log.Logger
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

type conns struct {
	common.Conn
}

func (c *conns) GetConns() []common.Conn {
	return []common.Conn{c.Conn}
}

func (h *HelloProtocol) StartClient(conn common.Conn) {
	h.StartServer(&conns{Conn: conn})
}

func (h *HelloProtocol) StartServer(conns Conns) {
	h.running = true
	msg := new(common.Message)
	msg.Type = h.HelloProtocolMsgType_Send
	for h.running {
		time.Sleep(h.Interval)
		for _, conn := range conns.GetConns() {
			isTimeout, isTimeoutClose := h.checkTimeout(conn.Activate(), h.Timeout, h.TimeoutClose)
			if isTimeoutClose {
				if h.Logger.Writer() != nil {
					h.Logger.Println("timeout close id", conn.RemoteId())
				}
				_ = conn.Close()
				continue
			}
			if isTimeout {
				msg.SrcId = conn.LocalId()
				msg.DestId = conn.RemoteId()
				if err := conn.WriteMsg(msg); err != nil {
					if h.Logger.Writer() != nil {
						h.Logger.Println("err: send hello ASK pack fail destId", conn.RemoteId(), err)
					}
					_ = conn.Close()
					continue
				}
				if h.Logger.Writer() != nil {
					h.Logger.Println("send hello ASK pack destId", conn.RemoteId())
				}
			}
		}
	}
}

func (h *HelloProtocol) CustomHandle(ctx common.Context) (next bool) {
	switch ctx.Type() {
	case h.HelloProtocolMsgType_Send:
		if h.Logger.Writer() != nil {
			h.Logger.Println("receive hello ASK pack srcId", ctx.SrcId())
		}
		if err := ctx.CustomReply(h.HelloProtocolMsgType_Reply, nil); err != nil {
			if h.Logger.Writer() != nil {
				h.Logger.Println("err: reply hello ACK pack srcId", ctx.SrcId(), "err", err)
			}
		}
		return false
	case h.HelloProtocolMsgType_Reply:
		if h.Logger.Writer() != nil {
			h.Logger.Println("receive hello ACK pack srcId", ctx.SrcId())
		}
		return false
	default:
		return true
	}
}

func (h *HelloProtocol) Stop() {
	h.running = false
}

func (h *HelloProtocol) checkTimeout(activate int64, timeout, timeoutClose time.Duration) (isTimeout, isTimeoutClose bool) {
	duration := time.Now().UnixMilli() - activate
	if duration > timeoutClose.Milliseconds() {
		return false, true
	}
	if duration > timeout.Milliseconds() {
		return true, false
	}
	return
}

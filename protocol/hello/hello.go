package hello

import (
	"errors"
	"fmt"
	"github.com/Li-giegie/node/common"
	"io"
	"time"
)

type Conns interface {
	GetConns() []common.Conn
}

type HelloProtocol struct {
	running                         bool
	output                          io.Writer
	HelloProtocolMsgType_Send       uint8
	HelloProtocolMsgType_Reply      uint8
	Interval, Timeout, TimeoutClose time.Duration
}

func NewHelloProtocol(msgTypeSend, msgTypeReply uint8, interval, timeout, timeoutClose time.Duration, output io.Writer) *HelloProtocol {
	return &HelloProtocol{
		output:                     output,
		HelloProtocolMsgType_Send:  msgTypeSend,
		HelloProtocolMsgType_Reply: msgTypeReply,
		Interval:                   interval,
		Timeout:                    timeout,
		TimeoutClose:               timeoutClose,
	}
}

func (h *HelloProtocol) StartClient(conn common.Conn) (err error) {
	h.running = true
	c := conn.(*common.Connect)
	for h.running {
		time.Sleep(h.Interval)
		isTimeout, isTimeoutClose := h.checkTimeout(c.Activate(), h.Timeout, h.TimeoutClose)
		if isTimeoutClose {
			if h.output != nil {
				h.output.Write([]byte("HelloProtocol: timeout close\n"))
			}
			return errors.New("timeout close")
		}
		if isTimeout {
			msg := c.MsgController.DefaultMsg()
			msg.Type = h.HelloProtocolMsgType_Send
			msg.SrcId = c.LocalId()
			msg.DestId = c.RemoteId()
			if err = c.WriteMsg(msg); err != nil {
				if h.output != nil {
					fmt.Fprintf(h.output, "HelloProtocol err: send hello pack fail %s\n", err.Error())
				}
				return err
			}
			c.MsgController.RecycleMsg(msg)
			if h.output != nil {
				h.output.Write([]byte("HelloProtocol: send hello pack\n"))
			}
		}
	}
	return nil
}

func (h *HelloProtocol) StartServer(conns Conns) {
	h.running = true
	for h.running {
		time.Sleep(h.Interval)
		for _, conn := range conns.GetConns() {
			c := conn.(*common.Connect)
			if c == nil {
				continue
			}
			isTimeout, isTimeoutClose := h.checkTimeout(c.Activate(), h.Timeout, h.TimeoutClose)
			if isTimeoutClose {
				if h.output != nil {
					fmt.Fprintf(h.output, "HelloProtocol: timeout close id=%d\n", conn.LocalId())
				}
				continue
			}
			if isTimeout {
				msg := c.MsgController.DefaultMsg()
				msg.Type = h.HelloProtocolMsgType_Send
				msg.SrcId = c.LocalId()
				msg.DestId = c.RemoteId()
				if err := c.WriteMsg(msg); err != nil {
					if h.output != nil {
						fmt.Fprintf(h.output, "HelloProtocol err: send hello pack fail conn close id=%d err=%s\n", conn.LocalId(), err.Error())
					}
					_ = c.Close()
					continue
				}
				c.MsgController.RecycleMsg(msg)
				if h.output != nil {
					fmt.Fprintf(h.output, "HelloProtocol: send hello pack id=%d\n", conn.LocalId())
				}
			}
		}
	}
}

func (h *HelloProtocol) CustomHandle(ctx common.Context) (next bool) {
	switch ctx.Type() {
	case h.HelloProtocolMsgType_Send:
		if h.output != nil {
			fmt.Fprintf(h.output, "HelloProtocol: receive hello pack srcId=%d\n", ctx.SrcId())
		}
		ctx.CustomReply(h.HelloProtocolMsgType_Reply, nil)
		return false
	case h.HelloProtocolMsgType_Reply:
		if h.output != nil {
			fmt.Fprintf(h.output, "HelloProtocol: receive hello reply pack srcId=%d\n", ctx.SrcId())
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

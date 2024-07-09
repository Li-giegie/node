package protocol

import (
	"errors"
	"fmt"
	"github.com/Li-giegie/node/common"
	"io"
	"time"
)

var (
	HelloProtocolMsgType_Send  = GetMsgType()
	HelloProtocolMsgType_Reply = GetMsgType()
)

type HelloProtocol struct {
	running bool
	output  io.Writer
}

func (h *HelloProtocol) InitClient(conn common.Conn, interval, timeout, timeoutClose time.Duration, output io.Writer) (err error) {
	h.running = true
	h.output = output
	c := conn.(*common.Connect)
	for h.running {
		time.Sleep(interval)
		isTimeout, isTimeoutClose := h.checkTimeout(c.Activate(), timeout, timeoutClose)
		if isTimeoutClose {
			if output != nil {
				output.Write([]byte("HelloProtocol: timeout close\n"))
			}
			return errors.New("timeout close")
		}
		if isTimeout {
			msg := c.MsgController.DefaultMsg()
			msg.Type = HelloProtocolMsgType_Send
			msg.SrcId = c.LocalId()
			msg.DestId = c.RemoteId()
			if err = c.WriteMsg(msg); err != nil {
				if output != nil {
					fmt.Fprintf(output, "HelloProtocol err: send hello pack fail %s\n", err.Error())
				}
				return err
			}
			c.MsgController.RecycleMsg(msg)
			if output != nil {
				output.Write([]byte("HelloProtocol: send hello pack\n"))
			}
		}
	}
	return nil
}

func (h *HelloProtocol) InitServer(conns Conns, interval, timeout, timeoutClose time.Duration, output io.Writer) {
	h.running = true
	h.output = output
	for h.running {
		time.Sleep(interval)
		for _, conn := range conns.GetConns() {
			c := conn.(*common.Connect)
			if c == nil {
				continue
			}
			isTimeout, isTimeoutClose := h.checkTimeout(c.Activate(), timeout, timeoutClose)
			if isTimeoutClose {
				if output != nil {
					fmt.Fprintf(output, "HelloProtocol: timeout close id=%d\n", conn.LocalId())
				}
				continue
			}
			if isTimeout {
				msg := c.MsgController.DefaultMsg()
				msg.Type = HelloProtocolMsgType_Send
				msg.SrcId = c.LocalId()
				msg.DestId = c.RemoteId()
				if err := c.WriteMsg(msg); err != nil {
					if output != nil {
						fmt.Fprintf(output, "HelloProtocol err: send hello pack fail conn close id=%d err=%s\n", conn.LocalId(), err.Error())
					}
					_ = c.Close()
					continue
				}
				c.MsgController.RecycleMsg(msg)
				if output != nil {
					fmt.Fprintf(output, "HelloProtocol: send hello pack id=%d\n", conn.LocalId())
				}
			}
		}
	}
}

func (h *HelloProtocol) CustomHandle(ctx common.Context) (next bool) {
	switch ctx.Type() {
	case HelloProtocolMsgType_Send:
		if h.output != nil {
			fmt.Fprintf(h.output, "HelloProtocol: receive hello pack srcId=%d\n", ctx.SrcId())
		}
		ctx.CustomReply(HelloProtocolMsgType_Reply, nil)
		return false
	case HelloProtocolMsgType_Reply:
		if h.output != nil {
			fmt.Fprintf(h.output, "HelloProtocol: receive hello reply pack srcId=%d\n", ctx.SrcId())
		}
		return false
	default:
		return true
	}
}

func (h *HelloProtocol) Abort() {
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

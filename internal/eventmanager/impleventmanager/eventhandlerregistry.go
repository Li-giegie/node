package impleventmanager

import (
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/ctx"
	"github.com/Li-giegie/node/pkg/handler"
	"github.com/Li-giegie/node/pkg/message"
	"net"
)

func NewEventManager() *EventManager {
	return &EventManager{}
}

type handleEmpty struct {
	typ uint8
	handler.Handler
}

// EventManager 对连接的生命周期进行了更细的划分，提供增加和删除处理器的功能
type EventManager struct {
	defaultOnAccept  handler.OnAcceptFunc
	defaultOnConnect handler.OnConnectFunc
	defaultOnMessage handler.OnMessageFunc
	defaultOnClose   handler.OnCloseFunc
	others           []*handleEmpty
}

func (c *EventManager) OnAccept(callback handler.OnAcceptFunc) {
	c.defaultOnAccept = callback
}

func (c *EventManager) OnConnect(callback handler.OnConnectFunc) {
	c.defaultOnConnect = callback
}

func (c *EventManager) OnMessage(callback handler.OnMessageFunc) {
	c.defaultOnMessage = callback
}

func (c *EventManager) OnClose(callback handler.OnCloseFunc) {
	c.defaultOnClose = callback
}

// Register 注册OnMessage事件ctx.Type()为指定typ的的Handler
func (c *EventManager) Register(typ uint8, h handler.Handler) bool {
	for _, other := range c.others {
		if other.typ == typ {
			return false
		}
	}
	c.others = append(c.others, &handleEmpty{typ: typ, Handler: h})
	return true
}

// Deregister 注销typ
func (c *EventManager) Deregister(typ uint8) bool {
	index := -1
	for i, other := range c.others {
		if other.typ == typ {
			index = i
			break
		}
	}
	if index >= 0 {
		c.others = append(c.others[:index], c.others[index+1:]...)
		return true
	}
	return false
}

func (c *EventManager) CallOnAccept(conn net.Conn) bool {
	if c.defaultOnAccept != nil {
		if !c.defaultOnAccept(conn) {
			return false
		}
	}
	for _, other := range c.others {
		if !other.OnAccept(conn) {
			return false
		}
	}
	return true
}

func (c *EventManager) CallOnConnect(conn conn.Conn) {
	if c.defaultOnConnect != nil {
		c.defaultOnConnect(conn)
	}
	for _, other := range c.others {
		other.OnConnect(conn)
	}
}

func (c *EventManager) CallOnMessage(ctx ctx.Context) {
	typ := ctx.Type()
	if typ == message.MsgType_Default {
		if c.defaultOnMessage != nil {
			c.defaultOnMessage(ctx)
		}
		return
	}
	for _, other := range c.others {
		if other.typ == typ {
			other.OnMessage(ctx)
			return
		}
	}
	_ = ctx.Response(message.StateCode_MessageTypeInvalid, nil)
}

func (c *EventManager) CallOnClose(conn conn.Conn, err error) {
	if c.defaultOnClose != nil {
		c.defaultOnClose(conn, err)
	}
	for _, other := range c.others {
		other.OnClose(conn, err)
	}
}

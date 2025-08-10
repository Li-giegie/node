package client

import (
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/reply"
)

type Handler interface {
	// OnMessage OnConnect之后每次收到请求时的回调函数，同步调用
	OnMessage(r *reply.Reply, m *message.Message)
	// OnClose 连接被关闭后的回调函数，同步调用
	OnClose(conn *conn.Conn, err error)
}

var Default Manager

type (
	OnMessageFunc func(r *reply.Reply, m *message.Message) (next bool)
	OnCloseFunc   func(conn *conn.Conn, err error) (next bool)
)

// Manager 处理器管理器
type Manager struct {
	onCloseFunc []OnCloseFunc
	handlers    map[uint8][]OnMessageFunc
}

func (m *Manager) AddOnMessage(fn ...OnMessageFunc) {
	for _, f := range fn {
		m.AddOnMessageWithType(message.MsgType_Default, f)
	}
}

func (m *Manager) AddOnMessageWithType(typ uint8, fn OnMessageFunc) {
	if m.handlers == nil {
		m.handlers = make(map[uint8][]OnMessageFunc)
	}
	m.handlers[typ] = append(m.handlers[typ], fn)
}

func (m *Manager) AddOnClose(fn ...OnCloseFunc) {
	m.onCloseFunc = append(m.onCloseFunc, fn...)
}

func (m *Manager) OnMessage(w *reply.Reply, msg *message.Message) {
	h := m.handlers[msg.Type]
	if h != nil {
		for _, fn := range h {
			fn(w, msg)
		}
	} else {
		w.Write(message.StateCode_MessageTypeInvalid, nil)
	}
}

func (m *Manager) OnClose(c *conn.Conn, err error) {
	for _, fn := range m.onCloseFunc {
		if !fn(c, err) {
			return
		}
	}
}

func OnMessage(fn ...OnMessageFunc) {
	Default.AddOnMessage(fn...)
}
func OnMessageType(typ uint8, fn OnMessageFunc) {
	Default.AddOnMessageWithType(typ, fn)
}
func OnClose(fn ...OnCloseFunc) {
	Default.AddOnClose(fn...)
}

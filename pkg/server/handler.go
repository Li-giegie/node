package server

import (
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/reply"
	"net"
)

type Handler interface {
	// OnAccept net.Listen.Accept之后第一个回调函数，同步调用
	OnAccept(conn net.Conn) (isClose bool)
	// OnConnect OnAccept之后的回调函数，同步调用
	OnConnect(conn *conn.Conn)
	// OnMessage OnConnect之后每次收到请求时的回调函数，同步调用
	OnMessage(r *reply.Reply, m *message.Message)
	// OnClose 连接被关闭后的回调函数，同步调用
	OnClose(conn *conn.Conn, err error)
}

var Default Manager

type (
	OnAcceptFunc  func(conn net.Conn) (next bool)
	OnConnectFunc func(conn *conn.Conn) (next bool)
	OnMessageFunc func(r *reply.Reply, m *message.Message) (next bool)
	OnCloseFunc   func(conn *conn.Conn, err error) (next bool)
)

// Manager 处理器管理器
type Manager struct {
	onAcceptFunc  []OnAcceptFunc
	onConnectFunc []OnConnectFunc
	onCloseFunc   []OnCloseFunc
	handlers      map[uint8][]OnMessageFunc
}

func (m *Manager) AddOnAccept(fn ...OnAcceptFunc) {
	m.onAcceptFunc = append(m.onAcceptFunc, fn...)
}

func (m *Manager) AddOnConnect(fn ...OnConnectFunc) {
	m.onConnectFunc = append(m.onConnectFunc, fn...)
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

func (m *Manager) OnAccept(c net.Conn) bool {
	for _, fn := range m.onAcceptFunc {
		if !fn(c) {
			return false
		}
	}
	return true
}

func (m *Manager) OnConnect(c *conn.Conn) {
	for _, fn := range m.onConnectFunc {
		if !fn(c) {
			return
		}
	}
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

func OnAccept(fn ...OnAcceptFunc) {
	Default.AddOnAccept(fn...)
}
func OnConnect(fn ...OnConnectFunc) {
	Default.AddOnConnect(fn...)
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

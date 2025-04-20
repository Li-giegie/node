package handlemanager

import (
	"fmt"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/handler"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"net"
)

// HandlerManager 处理器管理器
type HandlerManager struct {
	OnAcceptFunc  func(conn net.Conn) (next bool)
	OnConnectFunc func(conn conn.Conn) (next bool)
	OnMessageFunc func(w responsewriter.ResponseWriter, m *message.Message) (next bool)
	OnCloseFunc   func(conn conn.Conn, err error) (next bool)
	handlers      map[uint8]handler.Handler
}

func (m *HandlerManager) Register(typ uint8, h handler.Handler) {
	if m.handlers == nil {
		m.handlers = make(map[uint8]handler.Handler)
	}
	if _, ok := m.handlers[typ]; ok {
		panic(fmt.Sprintf("Register failed type %d exists\n", typ))
	}
	m.handlers[typ] = h
}

func (m *HandlerManager) Deregister(typ uint8) {
	delete(m.handlers, typ)
}

func (m *HandlerManager) OnAccept(acceptFunc func(conn net.Conn) (next bool)) {
	if m.OnAcceptFunc != nil {
		panic("OnAcceptFunc failed acceptFunc exists")
	}
	m.OnAcceptFunc = acceptFunc
}

func (m *HandlerManager) OnConnect(handlerFunc func(conn conn.Conn) (next bool)) {
	if m.OnConnectFunc != nil {
		panic("OnConnectFunc failed OnConnect exists")
	}
	m.OnConnectFunc = handlerFunc
}
func (m *HandlerManager) OnMessage(messageFunc func(w responsewriter.ResponseWriter, m *message.Message) (next bool)) {
	if m.OnMessageFunc != nil {
		panic("OnMessageFunc failed OnMessage exists")
	}
	m.OnMessageFunc = messageFunc
}
func (m *HandlerManager) OnClose(closeFunc func(conn conn.Conn, err error) (next bool)) {
	if m.OnCloseFunc != nil {
		panic("OnCloseFunc failed OnClose exists")
	}
	m.OnCloseFunc = closeFunc
}

func (m *HandlerManager) CallOnAccept(c net.Conn) bool {
	if m.OnAcceptFunc != nil {
		if !m.OnAcceptFunc(c) {
			return false
		}
	}
	for _, e := range m.handlers {
		if !e.OnAccept(c) {
			return false
		}
	}
	return true
}

func (m *HandlerManager) CallOnConnect(c conn.Conn) {
	if m.OnConnectFunc != nil {
		if !m.OnConnectFunc(c) {
			return
		}
	}
	for _, e := range m.handlers {
		e.OnConnect(c)
	}
}

func (m *HandlerManager) CallOnMessage(w responsewriter.ResponseWriter, msg *message.Message) {
	if m.OnMessageFunc != nil {
		if !m.OnMessageFunc(w, msg) {
			return
		}
	}
	h := m.handlers[msg.Type]
	if h != nil {
		h.OnMessage(w, msg)
	} else {
		w.Response(message.StateCode_MessageTypeInvalid, nil)
	}
}

func (m *HandlerManager) CallOnClose(c conn.Conn, err error) {
	if m.OnCloseFunc != nil {
		if !m.OnCloseFunc(c, err) {
			return
		}
	}
	for _, e := range m.handlers {
		e.OnClose(c, err)
	}
}

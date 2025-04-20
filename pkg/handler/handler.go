package handler

import (
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"net"
)

// Handler 处理器接口
type Handler interface {
	// OnAccept accept 行后的回到，allow是否允许接受连接，nil值默认接受
	OnAccept(conn net.Conn) (allow bool)
	// OnConnect 连接通过基础认证正式建立后的回调
	OnConnect(conn conn.Conn)
	// OnMessage 收到消息后的回调
	OnMessage(r responsewriter.ResponseWriter, msg *message.Message)
	// OnClose 连接关闭后的回调
	OnClose(conn conn.Conn, err error)
}

type (
	OnAcceptFunc  func(conn net.Conn) (allow bool)
	OnConnectFunc func(conn conn.Conn)
	OnMessageFunc func(r responsewriter.ResponseWriter, m *message.Message)
	OnCloseFunc   func(conn conn.Conn, err error)
)

// Empty 缺省Handler
type Empty struct{}

func (c *Empty) OnAccept(conn net.Conn) (allow bool)                           { return true }
func (c *Empty) OnConnect(conn conn.Conn)                                      {}
func (c *Empty) OnMessage(r responsewriter.ResponseWriter, m *message.Message) {}
func (c *Empty) OnClose(conn conn.Conn, err error)                             {}

// Default 缺省的处理器
type Default struct {
	OnAcceptFunc
	OnConnectFunc
	OnMessageFunc
	OnCloseFunc
}

func (h *Default) OnAccept(conn net.Conn) (allow bool) {
	if h.OnAcceptFunc != nil {
		return h.OnAcceptFunc(conn)
	}
	return true
}

func (h *Default) OnConnect(conn conn.Conn) {
	if h.OnConnectFunc != nil {
		h.OnConnectFunc(conn)
	}
}
func (h *Default) OnMessage(r responsewriter.ResponseWriter, m *message.Message) {
	if h.OnMessageFunc != nil {
		h.OnMessageFunc(r, m)
	}
}
func (h *Default) OnClose(conn conn.Conn, err error) {
	if h.OnCloseFunc != nil {
		h.OnCloseFunc(conn, err)
	}
}

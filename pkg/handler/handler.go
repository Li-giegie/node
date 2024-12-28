package handler

import (
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/ctx"
	"net"
)

type Handler interface {
	// OnAccept accept 行后的回到，allow是否允许接受连接，nil值默认接受
	OnAccept(conn net.Conn) (allow bool)
	// OnConnect 连接通过基础认证正式建立后的回调
	OnConnect(conn conn.Conn)
	// OnMessage 收到消息后的回调
	OnMessage(ctx ctx.Context)
	// OnClose 连接关闭后的回调
	OnClose(conn conn.Conn, err error)
}

type (
	OnAcceptFunc  func(conn net.Conn) (allow bool)
	OnConnectFunc func(conn conn.Conn)
	OnMessageFunc func(ctx ctx.Context)
	OnCloseFunc   func(conn conn.Conn, err error)
)

// EmptyHandler 缺省Handler
type EmptyHandler struct{}

func (c *EmptyHandler) OnAccept(conn net.Conn) (allow bool) { return true }

func (c *EmptyHandler) OnConnect(conn conn.Conn) {}

func (c *EmptyHandler) OnMessage(ctx ctx.Context) {}

func (c *EmptyHandler) OnClose(conn conn.Conn, err error) {}

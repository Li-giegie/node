package net

import (
	"github.com/Li-giegie/node/iface"
	"net"
)

func NewConnectionLifecycle() *ConnectionLifecycle {
	return new(ConnectionLifecycle)
}

type ConnectionLifecycle struct {
	onAcceptFunc  iface.OnAcceptFunc
	onConnectFunc iface.OnConnectFunc
	onMessageFunc iface.OnMessageFunc
	onCloseFunc   iface.OnCloseFunc
}

func (c *ConnectionLifecycle) OnAccept(callback iface.OnAcceptFunc) {
	c.onAcceptFunc = callback
}

func (c *ConnectionLifecycle) OnConnect(callback iface.OnConnectFunc) {
	c.onConnectFunc = callback
}

func (c *ConnectionLifecycle) OnMessage(callback iface.OnMessageFunc) {
	c.onMessageFunc = callback
}

func (c *ConnectionLifecycle) OnClose(callback iface.OnCloseFunc) {
	c.onCloseFunc = callback
}

func (c *ConnectionLifecycle) CallOnAccept(conn net.Conn) bool {
	if c.onAcceptFunc != nil {
		return c.onAcceptFunc(conn)
	}
	return true
}

func (c *ConnectionLifecycle) CallOnConnect(conn *Conn) {
	if c.onConnectFunc != nil {
		c.onConnectFunc(conn)
	}
}

func (c *ConnectionLifecycle) CallOnMessage(ctx *Context) {
	if c.onMessageFunc != nil {
		c.onMessageFunc(ctx)
	}
}

func (c *ConnectionLifecycle) CallOnClose(conn *Conn, err error) {
	if c.onCloseFunc != nil {
		c.onCloseFunc(conn, err)
	}
}

package node

import (
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	"net"
	"sync"
)

func NewHandler(node iface.ConnectionLifecycleCallback) *Handler {
	h := Handler{
		handler: map[uint8]iface.ConnectionLifecycle{},
	}
	node.OnAccept(h.callOnAccept)
	node.OnConnect(h.callOnConnect)
	node.OnMessage(h.callOnMessage)
	node.OnClose(h.callOnClose)
	return &h
}

// Handler 对连接的生命周期进行了更细的划分，提供增加和删除处理器的功能
type Handler struct {
	handler     map[uint8]iface.ConnectionLifecycle
	handlerLock sync.RWMutex
	beforeHandler
}

func (c *Handler) OnAccept(callback iface.OnAcceptFunc) {
	c.handlerLock.Lock()
	defer c.handlerLock.Unlock()
	handle, ok := c.handler[message.MsgType_Default]
	if !ok {
		handle = &DefaultHandler{OnAcceptFunc: callback}
		c.handler[message.MsgType_Default] = handle
		return
	}
	handle.(*DefaultHandler).OnAcceptFunc = callback
}

func (c *Handler) OnConnect(callback iface.OnConnectFunc) {
	c.handlerLock.Lock()
	defer c.handlerLock.Unlock()
	handle, ok := c.handler[message.MsgType_Default]
	if !ok {
		handle = &DefaultHandler{OnConnectFunc: callback}
		c.handler[message.MsgType_Default] = handle
		return
	}
	handle.(*DefaultHandler).OnConnectFunc = callback
}

func (c *Handler) OnMessage(callback iface.OnMessageFunc) {
	c.handlerLock.Lock()
	defer c.handlerLock.Unlock()
	handle, ok := c.handler[message.MsgType_Default]
	if !ok {
		handle = &DefaultHandler{OnMessageFunc: callback}
		c.handler[message.MsgType_Default] = handle
		return
	}
	handle.(*DefaultHandler).OnMessageFunc = callback
}

func (c *Handler) OnClose(callback iface.OnCloseFunc) {
	c.handlerLock.Lock()
	defer c.handlerLock.Unlock()
	handle, ok := c.handler[message.MsgType_Default]
	if !ok {
		handle = &DefaultHandler{OnCloseFunc: callback}
		c.handler[message.MsgType_Default] = handle
		return
	}
	handle.(*DefaultHandler).OnCloseFunc = callback
}

// Register 注册OnMessage事件ctx.Type()为指定typ的的Handler
func (c *Handler) Register(typ uint8, lifecycle iface.ConnectionLifecycle) bool {
	if lifecycle == nil {
		return false
	}
	c.handlerLock.Lock()
	defer c.handlerLock.Unlock()
	_, ok := c.handler[typ]
	if ok {
		return false
	}
	c.handler[typ] = lifecycle
	return true
}

// Deregister 注销typ
func (c *Handler) Deregister(typ uint8) bool {
	c.handlerLock.Lock()
	_, ok := c.handler[typ]
	if ok {
		delete(c.handler, typ)
	}
	c.handlerLock.Unlock()
	return ok
}

func (c *Handler) callOnAccept(conn net.Conn) bool {
	if c.beforeAcceptFunc != nil && !c.beforeAcceptFunc(conn) {
		return false
	}
	c.handlerLock.RLock()
	defer c.handlerLock.RUnlock()
	for _, lifecycle := range c.handler {
		if !lifecycle.OnAccept(conn) {
			return false
		}
	}
	return true
}

func (c *Handler) callOnConnect(conn iface.Conn) {
	if c.beforeConnectFunc != nil && !c.beforeConnectFunc(conn) {
		return
	}
	c.handlerLock.RLock()
	defer c.handlerLock.RUnlock()
	for _, lifecycle := range c.handler {
		lifecycle.OnConnect(conn)
	}
}

func (c *Handler) callOnMessage(ctx iface.Context) {
	if c.beforeMessageFunc != nil && !c.beforeMessageFunc(ctx) {
		return
	}
	c.handlerLock.RLock()
	handle, ok := c.handler[ctx.Type()]
	c.handlerLock.RUnlock()
	if !ok {
		_ = ctx.Response(message.StateCode_MessageTypeInvalid, nil)
		return
	}
	handle.OnMessage(ctx)
}

func (c *Handler) callOnClose(conn iface.Conn, err error) {
	if c.beforeCloseFunc != nil && !c.beforeCloseFunc(conn, err) {
		return
	}
	c.handlerLock.RLock()
	defer c.handlerLock.RUnlock()
	for _, lifecycle := range c.handler {
		lifecycle.OnClose(conn, err)
	}
}

type beforeHandler struct {
	beforeAcceptFunc  func(conn net.Conn) (next bool)
	beforeConnectFunc func(conn iface.Conn) (next bool)
	beforeMessageFunc func(ctx iface.Context) (next bool)
	beforeCloseFunc   func(conn iface.Conn, err error) (next bool)
}

func (b *beforeHandler) OnBeforeAccept(callback func(conn net.Conn) (next bool)) {
	b.beforeAcceptFunc = callback
}

func (b *beforeHandler) OnBeforeConnect(callback func(conn iface.Conn) (next bool)) {
	b.beforeConnectFunc = callback
}

func (b *beforeHandler) OnBeforeMessage(callback func(ctx iface.Context) (next bool)) {
	b.beforeMessageFunc = callback
}

func (b *beforeHandler) OnBeforeClose(callback func(conn iface.Conn, err error) (next bool)) {
	b.beforeCloseFunc = callback
}

// DefaultHandler 缺省Handler
type DefaultHandler struct {
	iface.OnAcceptFunc
	iface.OnConnectFunc
	iface.OnMessageFunc
	iface.OnCloseFunc
}

func (c *DefaultHandler) OnAccept(conn net.Conn) (allow bool) {
	if c.OnAcceptFunc != nil {
		return c.OnAcceptFunc(conn)
	}
	return true
}

func (c *DefaultHandler) OnConnect(conn iface.Conn) {
	if c.OnConnectFunc != nil {
		c.OnConnectFunc(conn)
	}
}

func (c *DefaultHandler) OnMessage(ctx iface.Context) {
	if c.OnMessageFunc != nil {
		c.OnMessageFunc(ctx)
	}
}

func (c *DefaultHandler) OnClose(conn iface.Conn, err error) {
	if c.OnCloseFunc != nil {
		c.OnCloseFunc(conn, err)
	}
}

// EmptyHandler 缺省Handler
type EmptyHandler struct{}

func (c *EmptyHandler) OnAccept(conn net.Conn) (allow bool) { return true }

func (c *EmptyHandler) OnConnect(conn iface.Conn) {}

func (c *EmptyHandler) OnMessage(ctx iface.Context) {}

func (c *EmptyHandler) OnClose(conn iface.Conn, err error) {}

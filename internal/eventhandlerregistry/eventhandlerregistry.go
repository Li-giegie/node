package eventhandlerregistry

import "github.com/Li-giegie/node/pkg/handler"

type EventHandlerRegistry interface {
	OnAccept(callback handler.OnAcceptFunc)
	OnConnect(callback handler.OnConnectFunc)
	OnMessage(callback handler.OnMessageFunc)
	OnClose(callback handler.OnCloseFunc)
	Register(typ uint8, h handler.Handler) bool
	Deregister(typ uint8) bool
}

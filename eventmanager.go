package node

import (
	"github.com/Li-giegie/node/iface"
	nodeNet "github.com/Li-giegie/node/net"
)

type eventManager struct {
	onConnects         []func(conn iface.Conn)
	onMessages         []func(ctx iface.Context)
	onProtocolMessages map[uint8][]func(ctx iface.Context)
	onRouteMessages    []func(ctx iface.Context)
	onCloses           []func(conn iface.Conn, err error) // 连接被关闭调用
}

func (s *eventManager) onConnect(conn iface.Conn) {
	for _, callback := range s.onConnects {
		callback(conn)
	}
}

func (s *eventManager) onMessage(ctx *nodeNet.Context) {
	for _, callback := range s.onMessages {
		callback(ctx)
		if !ctx.Next {
			return
		}
	}
}

func (s *eventManager) onProtocolMessage(ctx *nodeNet.Context) {
	callbacks, ok := s.onProtocolMessages[ctx.Type()]
	if !ok {
		_ = ctx.ReplyError(nodeNet.ErrMsgTypeInvalid, nil)
		return
	}
	for _, callback := range callbacks {
		callback(ctx)
		if !ctx.Next {
			return
		}
	}
}

func (s *eventManager) onClose(conn iface.Conn, err error) {
	for _, callback := range s.onCloses {
		callback(conn, err)
	}
}

func (s *eventManager) onRouteMessage(ctx *nodeNet.Context) {
	if len(s.onRouteMessages) == 0 {
		_ = ctx.ReplyError(nodeNet.ErrNodeNotExist, nil)
		return
	}
	for _, callback := range s.onRouteMessages {
		callback(ctx)
		if !ctx.Next {
			return
		}
	}
}

func (s *eventManager) AddOnConnect(callback iface.OnConnectFunc) {
	s.onConnects = append(s.onConnects, callback)
}

func (s *eventManager) AddOnMessage(callback iface.OnMessageFunc) {
	s.onMessages = append(s.onMessages, callback)
}

func (s *eventManager) AddOnProtocolMessage(typ uint8, callback iface.OnProtocolMessage) {
	if s.onProtocolMessages == nil {
		s.onProtocolMessages = map[uint8][]func(ctx iface.Context){}
	}
	callbacks := s.onProtocolMessages[typ]
	callbacks = append(callbacks, callback)
	s.onProtocolMessages[typ] = callbacks
}

func (s *eventManager) AddOnClose(callback iface.OnCloseFunc) {
	s.onCloses = append(s.onCloses, callback)
}

func (s *eventManager) AddOnRouteMessage(callback iface.OnRouteMessageFunc) {
	s.onRouteMessages = append(s.onRouteMessages, callback)
}

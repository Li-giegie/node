package node

import (
	"github.com/Li-giegie/node/iface"
	nodeNet "github.com/Li-giegie/node/net"
)

type connectionEvent struct {
	onConnects        []func(conn iface.Conn)
	onMessages        []func(ctx iface.Context)
	onCustomMessages  []func(ctx iface.Context)
	onForwardMessages []func(ctx iface.Context)
	onCloses          []func(conn iface.Conn, err error) // 连接被关闭调用
}

func (s *connectionEvent) onConnect(conn iface.Conn) {
	for _, callback := range s.onConnects {
		callback(conn)
	}
}

func (s *connectionEvent) onMessage(ctx iface.Context) {
	for _, callback := range s.onMessages {
		callback(ctx)
		if !ctx.Next() {
			return
		}
	}
}

func (s *connectionEvent) onCustomMessage(ctx iface.Context) {
	for _, callback := range s.onCustomMessages {
		callback(ctx)
		if !ctx.Next() {
			return
		}
	}
}

func (s *connectionEvent) onClose(conn iface.Conn, err error) {
	for _, callback := range s.onCloses {
		callback(conn, err)
	}
}

func (s *connectionEvent) onForwardMessage(ctx iface.Context) {
	if len(s.onForwardMessages) == 0 {
		_ = ctx.ReplyError(nodeNet.ErrNodeNotExist, nil)
		return
	}
	for _, callback := range s.onForwardMessages {
		callback(ctx)
		if !ctx.Next() {
			return
		}
	}
}

func (s *connectionEvent) AddOnConnect(callback func(conn iface.Conn)) {
	s.onConnects = append(s.onConnects, callback)
}

func (s *connectionEvent) AddOnMessage(callback func(conn iface.Context)) {
	s.onMessages = append(s.onMessages, callback)
}

func (s *connectionEvent) AddOnCustomMessage(callback func(conn iface.Context)) {
	s.onCustomMessages = append(s.onCustomMessages, callback)
}

func (s *connectionEvent) AddOnClose(callback func(conn iface.Conn, err error)) {
	s.onCloses = append(s.onCloses, callback)
}

func (s *connectionEvent) AddOnForwardMessage(callback func(conn iface.Context)) {
	s.onForwardMessages = append(s.onForwardMessages, callback)
}

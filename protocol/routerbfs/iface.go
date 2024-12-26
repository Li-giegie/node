package routerbfs

import "github.com/Li-giegie/node/iface"

type Protocol interface {
	ProtocolType() uint8
	iface.ConnectionLifecycle
}

type Node interface {
	Id() uint32
	GetAllConn() []iface.Conn
	GetConn(id uint32) (iface.Conn, bool)
	GetRouter() iface.Router
}

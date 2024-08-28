package node

import (
	"net"
)

type BridgeNode interface {
	Conn() net.Conn
	RemoteId() uint16
	Disconnection()
}

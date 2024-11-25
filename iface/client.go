package iface

import "net"

type Client interface {
	Id() uint32
	// Start 开启服务
	Start(conn net.Conn) (Conn, error)
	Handler
}

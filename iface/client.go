package iface

type Client interface {
	Id() uint32
	// Start 开启服务
	Start() error
	Handler
	Conn
}

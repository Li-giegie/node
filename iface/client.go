package iface

import (
	"crypto/tls"
	"net"
)

type Client interface {
	Id() uint32
	// Connect 发起连接 address 支持url格式例如 tcp://127.0.0.1:5555 = 127.0.0.1:5555，缺省协议默认tcp，config参数只能接受0个或者1个
	Connect(address string, config ...*tls.Config) (err error)
	// Start 开启服务
	Start(conn net.Conn) error
	Conn
	ConnectionLifecycleCallback
}

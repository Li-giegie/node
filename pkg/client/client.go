package client

import (
	"crypto/tls"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/handler"
	"net"
	"time"
)

type Client interface {
	Id() uint32
	// Connect 连接并异步开启服务 address 支持url格式例如 tcp://127.0.0.1:5555 = 127.0.0.1:5555，缺省协议默认tcp，config参数只能接受0个或者1个
	Connect(address string, config ...*tls.Config) (err error)
	// Start 阻塞开启服务
	Start(conn net.Conn) error
	// SetKeepalive 设置连接保活 interval、检查间隔时间，timeout 超时发送ASK时间、timeoutClose 超时关闭连接时间
	SetKeepalive(interval, timeout, timeoutClose time.Duration)
	conn.Conn

	OnAccept(callback handler.OnAcceptFunc)
	OnConnect(callback handler.OnConnectFunc)
	OnMessage(callback handler.OnMessageFunc)
	OnClose(callback handler.OnCloseFunc)
	Register(typ uint8, h handler.Handler) bool
	Deregister(typ uint8) bool
}

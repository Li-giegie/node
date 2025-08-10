package node

import (
	"context"
	"crypto/tls"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/router"
	"github.com/Li-giegie/node/pkg/server"
	"net"
)

type Server interface {
	// NodeId 当前节点ID
	NodeId() uint32
	// Serve 开启服务
	Serve(l net.Listener, h server.Handler) error
	// ListenAndServe 侦听并开启服务,address 支持url格式例如 tcp://127.0.0.1:5555 = 127.0.0.1:5555，缺省协议默认tcp
	ListenAndServe(address string, h server.Handler, conf ...*tls.Config) (err error)
	//Bridge 从当前节点桥接一个节点,组成一个更大的域，如果要完整启用该功能则需要开启节点动态发现协议
	Bridge(conn net.Conn, remoteId uint32, remoteAuthKey []byte) (err error)
	// GetConn 获取连接
	GetConn(id uint32) (*conn.Conn, bool)
	// GetAllConn 获取所有连接
	GetAllConn() []*conn.Conn
	LenConn() (n int)
	RangeConn(f func(conn *conn.Conn) bool)
	// GetRouter 获取路由
	GetRouter() router.Router
	RequestTo(ctx context.Context, dst uint32, data []byte) (int16, []byte, error)
	RequestTypeTo(ctx context.Context, typ uint8, dst uint32, data []byte) (int16, []byte, error)
	// RequestMessage 构建一个消息并发起请求，不要使用此方法发送消息，除非你知道自己在干什么，m的Id是Server内部维护的
	RequestMessage(ctx context.Context, msg *message.Message) (int16, []byte, error)
	SendTo(dst uint32, data []byte) error
	SendTypeTo(typ uint8, dst uint32, data []byte) error
	// SendMessage 构建一个消息并发送，不要使用此方法发送消息除非你知道自己在干什么，m的Id是Server内部维护的
	SendMessage(m *message.Message) error
	CreateMessageId() uint32
	CreateMessage(typ uint8, src uint32, dst uint32, data []byte) *message.Message
	RouteHop() uint8
	Close() error
}

func NewServer(c *server.Config) Server {
	return &server.Server{
		Id:                    c.Id,
		AuthKey:               c.AuthKey,
		AuthTimeout:           c.AuthTimeout,
		MaxMsgLen:             c.MaxMsgLen,
		WriterQueueSize:       c.WriterQueueSize,
		ReaderBufSize:         c.ReaderBufSize,
		WriterBufSize:         c.WriterBufSize,
		MaxConnections:        c.MaxConnections,
		SleepOnMaxConnections: c.SleepOnMaxConnections,
		KeepaliveInterval:     c.KeepaliveInterval,
		KeepaliveTimeout:      c.KeepaliveTimeout,
		KeepaliveTimeoutClose: c.KeepaliveTimeoutClose,
		MaxRouteHop:           c.MaxRouteHop,
	}
}

func NewServerOption(id uint32, opts ...server.Option) Server {
	return NewServer(server.DefaultConfig(append([]server.Option{server.WithId(id)}, opts...)...))
}

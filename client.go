package node

import (
	"context"
	"crypto/tls"
	"github.com/Li-giegie/node/pkg/client"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/message"
	"net"
)

type Client interface {
	NodeId() uint32
	// Connect 连接并异步开启服务 address 支持url格式例如 tcp://127.0.0.1:5555 = 127.0.0.1:5555，缺省协议默认tcp，config参数只能接受0个或者1个
	Connect(address string, h client.Handler, config ...*tls.Config) (err error)
	// Start 阻塞开启服务
	Start(conn net.Conn, h client.Handler) error
	State() client.State
	Send(data []byte) error
	SendMessage(m *message.Message) error
	SendTo(dst uint32, data []byte) error
	SendType(typ uint8, data []byte) error
	SendTypeTo(typ uint8, dst uint32, data []byte) error
	Request(ctx context.Context, data []byte) (int16, []byte, error)
	RequestTo(ctx context.Context, dst uint32, data []byte) (int16, []byte, error)
	RequestType(ctx context.Context, typ uint8, data []byte) (int16, []byte, error)
	RequestTypeTo(ctx context.Context, typ uint8, dst uint32, data []byte) (int16, []byte, error)
	RequestMessage(ctx context.Context, msg *message.Message) (int16, []byte, error)
	CreateMessage(typ uint8, src uint32, dst uint32, data []byte) *message.Message
	CreateMessageId() uint32
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	ConnType() conn.Type
}

func NewClient(c *client.Config) Client {
	return &client.Client{
		Id:              c.Id,
		RemoteID:        c.RemoteId,
		RemoteKey:       c.RemoteKey,
		AuthTimeout:     c.AuthTimeout,
		WriterQueueSize: c.WriterQueueSize,
		ReaderBufSize:   c.ReaderBufSize,
		WriterBufSize:   c.WriterBufSize,
	}
}

func NewClientOption(lid, rid uint32, opts ...client.Option) Client {
	return NewClient(client.DefaultConfig(append([]client.Option{client.WithId(lid), client.WithRemoteId(rid)}, opts...)...))
}

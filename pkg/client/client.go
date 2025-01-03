package client

import (
	"crypto/tls"
	"github.com/Li-giegie/node/pkg/client/implclient"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/handler"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"net"
	"time"
)

type Client interface {
	NodeId() uint32
	// Connect 连接并异步开启服务 address 支持url格式例如 tcp://127.0.0.1:5555 = 127.0.0.1:5555，缺省协议默认tcp，config参数只能接受0个或者1个
	Connect(address string, config ...*tls.Config) (err error)
	// Start 阻塞开启服务
	Start(conn net.Conn) error
	conn.Conn
	// OnAccept 注册全局OnAccept回调函数，net.Listen.Accept之后第一个回调函数，同步调用
	OnAccept(f func(conn net.Conn) (next bool))
	// OnConnect 注册全局OnConnect回调函数，OnAccept之后的回调函数，同步调用
	OnConnect(f func(conn conn.Conn) (next bool))
	// OnMessage 注册全局OnMessage回调函数，OnConnect之后每次收到请求时的回调函数，同步调用
	OnMessage(f func(r responsewriter.ResponseWriter, m *message.Message) (next bool))
	// OnClose 注册OnClose回调函数，连接被关闭后的回调函数，同步调用
	OnClose(f func(conn conn.Conn, err error) (next bool))
	// Register 注册实现了handler.Handler的处理接口，该接口的回调函数在OnAccept、OnConnect、OnMessage、OnClose之后被回调，同步调用
	Register(typ uint8, h handler.Handler)
	Deregister(typ uint8)
	State() bool
}

func NewClient(c *Config) Client {
	return &implclient.Client{
		Id:                    c.Id,
		RemoteID:              c.RemoteId,
		RemoteKey:             c.RemoteKey,
		AuthTimeout:           c.AuthTimeout,
		MaxMsgLen:             c.MaxMsgLen,
		WriterQueueSize:       c.WriterQueueSize,
		ReaderBufSize:         c.ReaderBufSize,
		WriterBufSize:         c.WriterBufSize,
		KeepaliveInterval:     c.KeepaliveInterval,
		KeepaliveTimeout:      c.KeepaliveTimeout,
		KeepaliveTimeoutClose: c.KeepaliveTimeoutClose,
	}
}

func NewClientOption(lid, rid uint32, opts ...Option) Client {
	return NewClient(DefaultConfig(append([]Option{WithId(lid), WithRemoteId(rid)}, opts...)...))
}

type Option func(*Config)

type Config struct {
	// 当前节点Id
	Id uint32
	// 远程节点Id
	RemoteId uint32
	// 远程节点Key
	RemoteKey []byte
	// 认证超时时长
	AuthTimeout time.Duration
	// 大于0时启用，收发消息最大长度，最大值0xffffffff
	MaxMsgLen uint32
	// 大于1时启用，并发请求或发送时，发出的消息不会被立即发出，而是会进入队列，直至队列缓冲区满，或者最后一个goroutine时才会将消息发出，如果消息要以最快的方式发出，那么请不要进入队列
	WriterQueueSize int
	// 读缓存区大小
	ReaderBufSize int
	// 大于64时启用，从队列读取后进入缓冲区，缓冲区大小
	WriterBufSize int
	// 连接保活检查时间间隔
	KeepaliveInterval time.Duration
	// 连接保活超时时间
	KeepaliveTimeout time.Duration
	// 连接保活最大超时次数
	KeepaliveTimeoutClose time.Duration
}

func DefaultConfig(opts ...Option) *Config {
	c := &Config{
		AuthTimeout:           time.Second * 6,
		MaxMsgLen:             0xffffff,
		WriterQueueSize:       1024,
		ReaderBufSize:         4096,
		WriterBufSize:         4096,
		KeepaliveInterval:     time.Second * 10,
		KeepaliveTimeout:      time.Second * 20,
		KeepaliveTimeoutClose: time.Minute,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithId(id uint32) Option {
	return func(c *Config) {
		c.Id = id
	}
}
func WithRemoteId(id uint32) Option {
	return func(c *Config) {
		c.RemoteId = id
	}
}
func WithRemoteKey(key []byte) Option {
	return func(config *Config) {
		config.RemoteKey = key
	}
}
func WithAuthTimeout(timeout time.Duration) Option {
	return func(config *Config) {
		config.AuthTimeout = timeout
	}
}
func WithMaxMsgLen(maxMsgLen uint32) Option {
	return func(config *Config) {
		config.MaxMsgLen = maxMsgLen
	}
}
func WithWriterQueueSize(writerQueueSize int) Option {
	return func(config *Config) {
		config.WriterQueueSize = writerQueueSize
	}
}
func WithReaderBufSize(bufferSize int) Option {
	return func(config *Config) {
		config.ReaderBufSize = bufferSize
	}
}
func WithWriterBufSize(bufferSize int) Option {
	return func(config *Config) {
		config.WriterBufSize = bufferSize
	}
}
func WithKeepaliveInterval(interval time.Duration) Option {
	return func(config *Config) {
		config.KeepaliveInterval = interval
	}
}
func WithKeepaliveTimeout(timeout time.Duration) Option {
	return func(config *Config) {
		config.KeepaliveTimeout = timeout
	}
}
func WithKeepaliveTimeoutClose(timeout time.Duration) Option {
	return func(config *Config) {
		config.KeepaliveTimeoutClose = timeout
	}
}

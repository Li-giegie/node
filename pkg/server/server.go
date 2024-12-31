package server

import (
	"context"
	"crypto/tls"
	"github.com/Li-giegie/node/internal"
	"github.com/Li-giegie/node/internal/connmanager/implconnmanager"
	"github.com/Li-giegie/node/internal/eventmanager/impleventmanager"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/handler"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/router"
	"github.com/Li-giegie/node/pkg/router/implrouter"
	"github.com/Li-giegie/node/pkg/server/implserver"
	"net"
	"sync"
	"time"
)

type Server interface {
	// NodeId 当前节点ID
	NodeId() uint32
	// Serve 开启服务
	Serve(l net.Listener) error
	// ListenAndServe 侦听并开启服务,address 支持url格式例如 tcp://127.0.0.1:5555 = 127.0.0.1:5555，缺省协议默认tcp
	ListenAndServe(address string, conf ...*tls.Config) (err error)
	//Bridge 从当前节点桥接一个节点,组成一个更大的域，如果要完整启用该功能则需要开启节点动态发现协议
	Bridge(conn net.Conn, remoteId uint32, remoteAuthKey []byte) (err error)
	// GetConn 获取连接
	GetConn(id uint32) (conn.Conn, bool)
	// GetAllConn 获取所有连接
	GetAllConn() []conn.Conn
	GetRouter() router.Router
	OnAccept(callback handler.OnAcceptFunc)
	OnConnect(callback handler.OnConnectFunc)
	OnMessage(callback handler.OnMessageFunc)
	OnClose(callback handler.OnCloseFunc)
	Register(typ uint8, h handler.Handler) bool
	Deregister(typ uint8) bool
	RequestTo(ctx context.Context, dst uint32, data []byte) (resp []byte, stateCode int16, err error)
	RequestTypeTo(ctx context.Context, typ uint8, dst uint32, data []byte) (resp []byte, stateCode int16, err error)
	RequestMessage(ctx context.Context, msg *message.Message) (resp []byte, stateCode int16, err error)
	SendTo(dst uint32, data []byte) error
	SendTypeTo(typ uint8, dst uint32, data []byte) error
	SendMessage(m *message.Message) error
	Close()
}

func NewServer(localId uint32, c *Config) Server {
	return &implserver.Server{
		Id:                    localId,
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
		MaxHop:                c.MaxHop,
		HashKey:               internal.Hash(c.AuthKey),
		RecvChan:              make(map[uint32]chan *message.Message),
		RecvLock:              sync.Mutex{},
		Router:                implrouter.NewRouter(),
		EventManager:          impleventmanager.NewEventManager(),
		ConnManager:           implconnmanager.NewConnManager(),
	}
}

func NewServerOption(localId uint32, opts ...Option) Server {
	return NewServer(localId, DefaultConfig(opts...))
}

func DefaultConfig(opts ...Option) *Config {
	c := &Config{
		MaxHop:                32,
		AuthTimeout:           time.Second * 6,
		MaxMsgLen:             0xffffff,
		WriterQueueSize:       128,
		ReaderBufSize:         4096,
		WriterBufSize:         4096,
		KeepaliveInterval:     time.Second * 20,
		KeepaliveTimeout:      time.Minute,
		KeepaliveTimeoutClose: time.Minute,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type Config struct {
	// 一条消息的最大转发次数
	MaxHop uint8
	// 节点认证Key
	AuthKey []byte
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
	// 大于0启用，最大连接数
	MaxConnections int
	// 超过最大连接休眠时长，MaxConns>0时有效
	SleepOnMaxConnections time.Duration
	// 连接保活检查时间间隔 > 0启用
	KeepaliveInterval time.Duration
	// 连接保活超时时间 > 0启用
	KeepaliveTimeout time.Duration
	// 连接保活最大超时次数
	KeepaliveTimeoutClose time.Duration
}

type Option func(*Config)

func WithAuthKey(key []byte) Option {
	return func(c *Config) {
		c.AuthKey = key
	}
}
func WithAuthTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.AuthTimeout = timeout
	}
}
func WithMaxMsgLen(maxMsgLen uint32) Option {
	return func(c *Config) {
		c.MaxMsgLen = maxMsgLen
	}
}
func WithWriterQueueSize(max int) Option {
	return func(c *Config) {
		c.WriterQueueSize = max
	}
}

func WithReaderBufSize(max int) Option {
	return func(c *Config) {
		c.ReaderBufSize = max
	}
}
func WithWriterBufSize(max int) Option {
	return func(c *Config) {
		c.WriterBufSize = max
	}
}
func WithMaxConnections(max int) Option {
	return func(c *Config) {
		c.MaxConnections = max
	}
}
func WithSleepOnMaxConnections(sleepOnMaxConnections time.Duration) Option {
	return func(c *Config) {
		c.SleepOnMaxConnections = sleepOnMaxConnections
	}
}
func WithKeepaliveInterval(keepaliveInterval time.Duration) Option {
	return func(c *Config) {
		c.KeepaliveInterval = keepaliveInterval
	}
}
func WithKeepaliveTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.KeepaliveTimeout = timeout
	}
}
func WithKeepaliveTimeoutClose(timeout time.Duration) Option {
	return func(c *Config) {
		c.KeepaliveTimeoutClose = timeout
	}
}

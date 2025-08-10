package server

import "time"

func DefaultConfig(opts ...Option) *Config {
	c := &Config{
		MaxRouteHop:           32,
		AuthTimeout:           time.Second * 6,
		MaxMsgLen:             0xffffff,
		WriterQueueSize:       128,
		ReaderBufSize:         4096,
		WriterBufSize:         4096,
		KeepaliveInterval:     time.Second * 20,
		KeepaliveTimeout:      time.Second * 40,
		KeepaliveTimeoutClose: time.Second * 120,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type Config struct {
	// 节点Id
	Id uint32
	// 一条消息的最大转发次数
	MaxRouteHop uint8
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

func WithId(id uint32) Option {
	return func(c *Config) {
		c.Id = id
	}
}

func WithMaxRouteHop(n uint8) Option {
	return func(c *Config) {
		c.MaxRouteHop = n
	}
}

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

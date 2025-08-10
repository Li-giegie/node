package client

import "time"

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
	// 大于1时启用，并发请求或发送时，发出的消息不会被立即发出，而是会进入队列，直至队列缓冲区满，或者最后一个goroutine时才会将消息发出，如果消息要以最快的方式发出，那么请不要进入队列
	WriterQueueSize int
	// 读缓存区大小
	ReaderBufSize int
	// 大于64时启用，从队列读取后进入缓冲区，缓冲区大小
	WriterBufSize int
}

func DefaultConfig(opts ...Option) *Config {
	c := &Config{
		AuthTimeout:     time.Second * 6,
		WriterQueueSize: 1024,
		ReaderBufSize:   4096,
		WriterBufSize:   4096,
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

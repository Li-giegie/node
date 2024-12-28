package common

import (
	"time"
)

type Identity struct {
	// Id 节点Id
	Id uint32
	// Key 认证的秘钥
	Key []byte
	// AuthTimeout 认证超时时间、超过时间没有认证成功断开连接
	AuthTimeout time.Duration
}

type Config struct {
	// 大于0时启用，收发消息最大长度，最大值0xffffffff
	MaxMsgLen uint32
	// 大于1时启用，并发请求或发送时，发出的消息不会被立即发出，而是会进入队列，直至队列缓冲区满，或者最后一个goroutine时才会将消息发出，如果消息要以最快的方式发出，那么请不要进入队列
	WriterQueueSize int
	// 读缓存区大小
	ReaderBufSize int
	// 大于64时启用，从队列读取后进入缓冲区，缓冲区大小
	WriterBufSize int
	// 大于0启用，最大连接数，客户端节点该字段无效
	MaxConns int
	// 小于等于0时panic 超过最大连接休眠时长，客户端节点该字段无效
	MaxConnSleep time.Duration
	// 是否开启连接保活
	Keepalive bool
	// 连接保活检查时间间隔
	KeepaliveInterval time.Duration
	// 连接保活超时时间
	KeepaliveTimeout time.Duration
	// 连接保活最大超时次数
	KeepaliveTimeoutClose time.Duration
}

var DefaultConfig = &Config{
	MaxMsgLen:       0xffffff,
	WriterQueueSize: 1024,
	ReaderBufSize:   4096,
	WriterBufSize:   4096,
	MaxConns:        0,
	MaxConnSleep:    time.Second * 5,
}

package node

import "time"

type Config struct {
	// 收发消息最大长度，最大值0xffffffff
	MaxMsgLen uint32
	// 消息并发写入队列，队列长度，服务端推荐>=cpu核心数
	WriterQueueSize int
	// 读缓存区大小
	ReaderBufSize int
	// 写缓冲区大小
	WriterBufSize int
	// 最大连接数 <= 0不限制,客户端节点该字段无效
	MaxConns int
	// 超过最大连接休眠时长，客户端节点该字段无效
	MaxConnSleep time.Duration
}

var defaultConfig = &Config{
	MaxMsgLen:       0xffffff,
	WriterQueueSize: 1024,
	ReaderBufSize:   4096,
	WriterBufSize:   4096,
	MaxConns:        0,
	MaxConnSleep:    time.Second * 5,
}

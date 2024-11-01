package net

import (
	"net"
	"time"
)

type ConnsLen interface {
	Len() int
}

type limitListener struct {
	MaxConns int
	// 超过最大连接数时，进入休眠的最大时间，按照步长递增
	MaxListenSleepTime time.Duration
	// 超过限制连接数量，递增休眠步长
	ListenStepTime time.Duration
	ConnsLen
	net.Listener
	isClose bool
}

func (l *limitListener) Accept() (conn net.Conn, err error) {
	count := l.ListenStepTime
	for l.MaxConns > 0 && l.ConnsLen.Len() >= l.MaxConns {
		time.Sleep(count)
		if count < l.MaxListenSleepTime {
			count += l.ListenStepTime
		}
	}
	conn, err = l.Listener.Accept()
	if err != nil && l.isClose {
		err = DEFAULT_ErrClosedListen
	}
	return
}

func (l *limitListener) Close() error {
	l.isClose = true
	return l.Listener.Close()
}

func NewLimitListener(l net.Listener, maxConns int, maxListenSleepTime, listenStepTime time.Duration, cl ConnsLen) net.Listener {
	return &limitListener{
		MaxConns:           maxConns,
		MaxListenSleepTime: maxListenSleepTime,
		ListenStepTime:     listenStepTime,
		ConnsLen:           cl,
		Listener:           l,
	}
}

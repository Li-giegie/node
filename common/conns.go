package common

import (
	"context"
	"sync"
)

type SrvConn interface {
	//Request 发起一个请求，得到一个响应
	Request(ctx context.Context, api uint16, data []byte) ([]byte, error)
	//Send 仅发送数据
	Send(api uint16, data []byte) (err error)
	Close() error
	State() ConnStateType
	//WriteMsg 在不需要响应的情况下，且仅发送到一个目的连接中时（类似单播），应优先使用Send方法，直接调用此方法不会重写目的id，对端收到后将丢弃该消息。如果一条消息想发送到多个连接中时（类似广播），可以类型断言成*common.Message,每次发送前修改destId为成对应连接的id
	WriteMsg(m Encoder) (int, error)
	Id() uint16
	//Activate 激活时间单位毫秒
	Activate() int64
}

type Conns struct {
	M map[uint16]Conn
	sync.RWMutex
}

func NewConns() *Conns {
	return &Conns{
		M:       make(map[uint16]Conn),
		RWMutex: sync.RWMutex{},
	}
}

func (s *Conns) Add(id uint16, conn Conn) bool {
	s.Lock()
	v, exist := s.M[id]
	if !exist || v.State() != ConnStateTypeOnConnect {
		s.M[id] = conn
		exist = true
	} else {
		exist = false
	}
	s.Unlock()
	return exist
}

func (s *Conns) Del(id uint16) {
	s.Lock()
	delete(s.M, id)
	s.Unlock()
}

func (s *Conns) GetConn(id uint16) (SrvConn, bool) {
	s.RLock()
	v, ok := s.M[id]
	s.RUnlock()
	return v, ok
}

func (s *Conns) GetConns() []SrvConn {
	s.RLock()
	var result = make([]SrvConn, 0, len(s.M))
	for _, conn := range s.M {
		result = append(result, conn)
	}
	s.RUnlock()
	return result
}

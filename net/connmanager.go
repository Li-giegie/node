package net

import (
	"github.com/Li-giegie/node/iface"
	"sync"
)

type ConnManager struct {
	m map[uint32]iface.Conn
	l sync.RWMutex
}

func NewConnManager() *ConnManager {
	return &ConnManager{
		m: make(map[uint32]iface.Conn),
		l: sync.RWMutex{},
	}
}

func (s *ConnManager) AddConn(id uint32, conn iface.Conn) bool {
	s.l.Lock()
	_, exist := s.m[id]
	if !exist {
		s.m[id] = conn
		exist = true
	} else {
		exist = false
	}
	s.l.Unlock()
	return exist
}

func (s *ConnManager) RemoveConn(id uint32) {
	s.l.Lock()
	delete(s.m, id)
	s.l.Unlock()
}

func (s *ConnManager) GetConn(id uint32) (iface.Conn, bool) {
	s.l.RLock()
	v, ok := s.m[id]
	s.l.RUnlock()
	return v, ok
}

func (s *ConnManager) GetAllConn() []iface.Conn {
	s.l.RLock()
	result := make([]iface.Conn, 0, len(s.m))
	for _, conn := range s.m {
		result = append(result, conn)
	}
	s.l.RUnlock()
	return result
}

func (s *ConnManager) Len() (n int) {
	s.l.RLock()
	n = len(s.m)
	s.l.RUnlock()
	return
}

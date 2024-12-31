package implconnmanager

import (
	"github.com/Li-giegie/node/pkg/conn"
	"sync"
)

type ConnManager struct {
	m map[uint32]conn.Conn
	l sync.RWMutex
}

func NewConnManager() *ConnManager {
	return &ConnManager{
		m: make(map[uint32]conn.Conn),
		l: sync.RWMutex{},
	}
}

func (s *ConnManager) AddConn(id uint32, conn conn.Conn) bool {
	s.l.Lock()
	_, exist := s.m[id]
	if !exist {
		s.m[id] = conn
	}
	s.l.Unlock()
	return !exist
}

func (s *ConnManager) RemoveConn(id uint32) {
	s.l.Lock()
	delete(s.m, id)
	s.l.Unlock()
}

func (s *ConnManager) GetConn(id uint32) (conn.Conn, bool) {
	s.l.RLock()
	v, ok := s.m[id]
	s.l.RUnlock()
	return v, ok
}

func (s *ConnManager) GetAllConn() []conn.Conn {
	s.l.RLock()
	result := make([]conn.Conn, 0, len(s.m))
	for _, conn := range s.m {
		result = append(result, conn)
	}
	s.l.RUnlock()
	return result
}

func (s *ConnManager) RangeConn(f func(conn conn.Conn) bool) {
	s.l.RLock()
	defer s.l.RUnlock()
	for _, conn := range s.m {
		if !f(conn) {
			return
		}
	}
}

func (s *ConnManager) LenConn() (n int) {
	s.l.RLock()
	n = len(s.m)
	s.l.RUnlock()
	return
}

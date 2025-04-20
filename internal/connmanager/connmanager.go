package connmanager

import (
	"github.com/Li-giegie/node/pkg/conn"
	"sync"
)

type ConnManager struct {
	m map[uint32]conn.Conn
	l sync.RWMutex
}

func (s *ConnManager) AddConn(c conn.Conn) bool {
	s.l.Lock()
	defer s.l.Unlock()
	if s.m == nil {
		s.m = make(map[uint32]conn.Conn)
	}
	_, exist := s.m[c.RemoteId()]
	if !exist {
		s.m[c.RemoteId()] = c
	}
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
	for _, c := range s.m {
		result = append(result, c)
	}
	s.l.RUnlock()
	return result
}

func (s *ConnManager) RangeConn(f func(conn conn.Conn) bool) {
	s.l.RLock()
	defer s.l.RUnlock()
	for _, c := range s.m {
		if !f(c) {
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

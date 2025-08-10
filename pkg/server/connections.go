package server

import (
	"github.com/Li-giegie/node/pkg/conn"
	"sync"
)

type connections struct {
	m map[uint32]*conn.Conn
	l sync.RWMutex
}

func (s *connections) AddConn(c *conn.Conn) bool {
	s.l.Lock()
	defer s.l.Unlock()
	if s.m == nil {
		s.m = make(map[uint32]*conn.Conn)
	}
	_, exist := s.m[c.RemoteId()]
	if !exist {
		s.m[c.RemoteId()] = c
	}
	return !exist
}

func (s *connections) RemoveConn(id uint32) {
	s.l.Lock()
	delete(s.m, id)
	s.l.Unlock()
}

func (s *connections) GetConn(id uint32) (*conn.Conn, bool) {
	s.l.RLock()
	v, ok := s.m[id]
	s.l.RUnlock()
	return v, ok
}

func (s *connections) GetAllConn() []*conn.Conn {
	s.l.RLock()
	result := make([]*conn.Conn, 0, len(s.m))
	for _, c := range s.m {
		result = append(result, c)
	}
	s.l.RUnlock()
	return result
}

func (s *connections) RangeConn(f func(conn *conn.Conn) bool) {
	s.l.RLock()
	defer s.l.RUnlock()
	for _, c := range s.m {
		if !f(c) {
			return
		}
	}
}

func (s *connections) LenConn() (n int) {
	s.l.RLock()
	n = len(s.m)
	s.l.RUnlock()
	return
}

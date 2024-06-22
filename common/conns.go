package common

import (
	"sync"
)

type Conns struct {
	m map[uint16]*Connect
	l sync.RWMutex
}

func NewConns() *Conns {
	return &Conns{
		m: make(map[uint16]*Connect),
		l: sync.RWMutex{},
	}
}

func (s *Conns) Add(id uint16, conn *Connect) bool {
	s.l.Lock()
	v, exist := s.m[id]
	if !exist || v.State() != ConnStateTypeOnConnect {
		s.m[id] = conn
		exist = true
	} else {
		exist = false
	}
	s.l.Unlock()
	return exist
}

func (s *Conns) Del(id uint16) {
	s.l.Lock()
	delete(s.m, id)
	s.l.Unlock()
}

func (s *Conns) GetConn(id uint16) (Conn, bool) {
	s.l.RLock()
	v, ok := s.m[id]
	s.l.RUnlock()
	return v, ok
}

func (s *Conns) GetConns() []Conn {
	s.l.RLock()
	var result = make([]Conn, 0, len(s.m))
	for _, conn := range s.m {
		result = append(result, conn)
	}
	s.l.RUnlock()
	return result
}

func (s *Conns) Len() (n int) {
	s.l.RLock()
	n = len(s.m)
	s.l.RUnlock()
	return
}

type emptyConns struct {
}

func (e emptyConns) GetConn(id uint16) (Conn, bool) {
	return nil, false
}

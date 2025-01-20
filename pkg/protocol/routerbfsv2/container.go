package routerbfsv2

import (
	"github.com/Li-giegie/node/pkg/conn"
	"sync"
)

type PeerTab struct {
	m map[uint32]*Conn
	l sync.RWMutex
}

func (t *PeerTab) Add(id uint32, conn *Conn) {
	t.l.Lock()
	defer t.l.Unlock()
	if t.m == nil {
		t.m = make(map[uint32]*Conn)
	}
	t.m[id] = conn
}

func (t *PeerTab) Remove(id uint32) {
	t.l.Lock()
	defer t.l.Unlock()
	if t.m == nil {
		return
	}
	delete(t.m, id)
}

func (t *PeerTab) Get(id uint32) (*Conn, bool) {
	t.l.RLock()
	defer t.l.RUnlock()
	c, ok := t.m[id]
	return c, ok
}

func (t *PeerTab) Len() int {
	t.l.RLock()
	defer t.l.RUnlock()
	return len(t.m)
}

func (t *PeerTab) Range(f func(uint32, *Conn) bool) {
	t.l.RLock()
	defer t.l.RUnlock()
	for k, v := range t.m {
		if !f(k, v) {
			return
		}
	}
}

type Conn struct {
	Conn     conn.Conn
	UpdateAt int64
}

type NodeTab struct {
	m map[uint32]*Conn
	l sync.RWMutex
}

func (n *NodeTab) Add(id uint32, conn *Conn) {
	n.l.Lock()
	defer n.l.Unlock()
	if n.m == nil {
		n.m = make(map[uint32]*Conn)
	}
	n.m[id] = conn
}
func (n *NodeTab) Remove(id uint32) {
	n.l.Lock()
	defer n.l.Unlock()
	if n.m == nil {
		return
	}
	delete(n.m, id)
}
func (n *NodeTab) Get(id uint32) (*Conn, bool) {
	n.l.RLock()
	defer n.l.RUnlock()
	c, ok := n.m[id]
	return c, ok
}
func (n *NodeTab) Len() int {
	n.l.RLock()
	defer n.l.RUnlock()
	return len(n.m)
}
func (n *NodeTab) Range(f func(uint32, *Conn) bool) {
	n.l.RLock()
	defer n.l.RUnlock()
	for k, v := range n.m {
		if !f(k, v) {
			return
		}
	}
}

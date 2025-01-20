package routerbfs

import (
	"encoding/json"
	"github.com/Li-giegie/node/pkg/conn"
	"sync"
)

func newNodeTable() *nodeTable {
	return &nodeTable{
		cache: make(map[uint32]map[uint32]struct{}),
	}
}

// nodeContainer 所有节点容器
type nodeTable struct {
	// root-node -> sub-node -> node-unixNao
	cache map[uint32]map[uint32]struct{}
	sync.RWMutex
}

func (tab *nodeTable) AddRoot(id uint32) bool {
	tab.Lock()
	defer tab.Unlock()
	_, ok := tab.cache[id]
	if !ok {
		tab.cache[id] = make(map[uint32]struct{})
	}
	return !ok
}

func (tab *nodeTable) RemoveRoot(id uint32) bool {
	tab.Lock()
	defer tab.Unlock()
	_, ok := tab.cache[id]
	if ok {
		delete(tab.cache, id)
	}
	return ok
}

func (tab *nodeTable) AddSub(rootId uint32, subId uint32) bool {
	tab.Lock()
	defer tab.Unlock()
	empty, ok := tab.cache[rootId]
	if !ok {
		tab.cache[rootId] = map[uint32]struct{}{subId: {}}
		return true
	}
	if _, ok = empty[subId]; !ok {
		empty[subId] = struct{}{}
	}
	return !ok
}

func (tab *nodeTable) Len() int {
	tab.RLock()
	defer tab.RUnlock()
	return len(tab.cache)
}

func (tab *nodeTable) DeleteSub(rootId uint32, subId uint32) bool {
	tab.Lock()
	defer tab.Unlock()
	empty, ok := tab.cache[rootId]
	if !ok {
		return false
	}
	_, ok = empty[subId]
	if !ok {
		return false
	}
	delete(empty, subId)
	if len(empty) == 0 {
		delete(tab.cache, rootId)
	}
	return true
}

func (tab *nodeTable) Range(f func(rootId uint32, empty map[uint32]struct{}) bool) {
	tab.RLock()
	defer tab.RUnlock()
	for rootId, empty := range tab.cache {
		if !f(rootId, empty) {
			return
		}
	}
}

func (tab *nodeTable) RangeSubNode(rootId uint32, callback func(map[uint32]struct{})) {
	tab.RLock()
	defer tab.RUnlock()
	empty, rootExist := tab.cache[rootId]
	if !rootExist {
		return
	}
	callback(empty)
	return
}

func (tab *nodeTable) GetSubNodes(rootId uint32) []uint32 {
	tab.RLock()
	defer tab.RUnlock()
	empty := tab.cache[rootId]
	result := make([]uint32, 0, len(empty))
	for u := range empty {
		result = append(result, u)
	}
	return result
}

func (tab *nodeTable) Encode() ([]byte, error) {
	tab.RLock()
	defer tab.RUnlock()
	return json.Marshal(tab.cache)
}

func newNeighborTable() *neighborTable {
	return &neighborTable{
		cache: make(map[uint32]*Conn),
	}
}

type neighborTable struct {
	cache map[uint32]*Conn
	l     sync.RWMutex
}

type Conn struct {
	conn.Conn
}

func (p *neighborTable) AddNeighbor(id uint32, conn conn.Conn) bool {
	p.l.Lock()
	defer p.l.Unlock()
	if p.cache[id] == nil {
		p.cache[id] = &Conn{
			Conn: conn,
		}
		return true
	}
	return false
}

func (p *neighborTable) GetNeighbor(id uint32) (*Conn, bool) {
	p.l.RLock()
	c, ok := p.cache[id]
	p.l.RUnlock()
	return c, ok
}

func (p *neighborTable) DeleteNeighbor(id uint32) bool {
	p.l.Lock()
	defer p.l.Unlock()
	if p.cache[id] != nil {
		delete(p.cache, id)
		return true
	}
	return false
}

func (p *neighborTable) RangeNeighbor(callback func(id uint32, conn *Conn) bool) {
	p.l.RLock()
	defer p.l.RUnlock()
	for u, c := range p.cache {
		if !callback(u, c) {
			return
		}
	}
}

func (p *neighborTable) LenNeighbor() int {
	p.l.RLock()
	l := len(p.cache)
	p.l.RUnlock()
	return l
}

func newCheckTable(valSize int) *checkTable {
	return &checkTable{
		empty:   make(map[uint32][]int64),
		valSize: valSize,
	}
}

type checkTable struct {
	empty   map[uint32][]int64
	valSize int
	sync.RWMutex
}

func (c *checkTable) Permit(id uint32, unixNano int64) bool {
	c.RLock()
	defer c.RUnlock()
	arr, ok := c.empty[id]
	if !ok {
		vals := make([]int64, c.valSize)
		vals[0] = unixNano
		c.empty[id] = vals
		return true
	}
	minIndex := 0
	minVal := arr[0]
	for i, v := range arr {
		if unixNano == v {
			return false
		}
		if v < minVal {
			minVal = v
			minIndex = i
		}
	}
	if unixNano < minVal {
		return false
	}
	arr[minIndex] = unixNano
	return true
}

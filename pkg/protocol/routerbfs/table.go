package routerbfs

import (
	"github.com/Li-giegie/node/pkg/conn"
	"sync"
)

func newNodeTable() *nodeTable {
	return &nodeTable{
		cache: make(map[uint32]*NodeTableEmpty),
	}
}

// nodeContainer 所有节点容器
type nodeTable struct {
	// root-node -> sub-node -> node-unixNao
	cache map[uint32]*NodeTableEmpty
	sync.RWMutex
}

type NodeTableEmpty struct {
	Cache map[uint32]conn.NodeType
}

func (tab *nodeTable) AddNode(rootId uint32, subId uint32, subType conn.NodeType) bool {
	tab.Lock()
	defer tab.Unlock()
	empty, ok := tab.cache[rootId]
	if !ok {
		tab.cache[rootId] = &NodeTableEmpty{
			Cache: map[uint32]conn.NodeType{
				subId: subType,
			},
		}
		return true
	}
	if empty.Cache[subId] == subType {
		return false
	}
	empty.Cache[subId] = subType
	return true
}

func (tab *nodeTable) RemoveRootNode(rootId uint32) bool {
	tab.Lock()
	defer tab.Unlock()
	_, ok := tab.cache[rootId]
	if ok {
		delete(tab.cache, rootId)
	}
	return ok
}

func (tab *nodeTable) RemoveNode(rootId uint32, subId uint32, subTyp conn.NodeType) bool {
	tab.Lock()
	defer tab.Unlock()
	empty, ok := tab.cache[rootId]
	if !ok {
		return false
	}
	sType, ok := empty.Cache[subId]
	if !ok {
		return false
	}
	delete(empty.Cache, subId)
	if len(empty.Cache) == 0 {
		delete(tab.cache, rootId)
	}
	if sType == conn.NodeTypeServer {
		subEmpty := tab.cache[subId]
		if subEmpty != nil {
			for u, nodeType := range subEmpty.Cache {
				if nodeType == conn.NodeTypeServer {
					if v := tab.cache[u]; v != nil {
						delete(v.Cache, subId)
						if len(v.Cache) == 0 {
							delete(tab.cache, u)
						}
					}
				}
			}
			delete(tab.cache, subId)
		}
	}
	return true
}

func (tab *nodeTable) Len() int {
	tab.RLock()
	defer tab.RUnlock()
	return len(tab.cache)
}

func (tab *nodeTable) Range(f func(rootId uint32, empty *NodeTableEmpty) bool) {
	tab.RLock()
	defer tab.RUnlock()
	for rootId, empty := range tab.cache {
		if !f(rootId, empty) {
			return
		}
	}
}

func (tab *nodeTable) RootList(filter ...uint32) *List {
	tab.RLock()
	defer tab.RUnlock()
	list := make(List, 0, len(tab.cache))
	var has bool
	for rootId := range tab.cache {
		has = false
		for _, u := range filter {
			if rootId == u {
				has = true
				break
			}
		}
		if !has {
			list = append(list, rootId)
		}
	}
	return &list
}

func (tab *nodeTable) RangeSubNode(rootId uint32, callback func(map[uint32]conn.NodeType)) {
	tab.RLock()
	defer tab.RUnlock()
	empty, rootExist := tab.cache[rootId]
	if !rootExist {
		return
	}
	callback(empty.Cache)
	return
}

func (tab *nodeTable) GetSubNodes(rootId uint32) []uint32 {
	tab.RLock()
	defer tab.RUnlock()
	empty := tab.cache[rootId]
	if empty == nil {
		return nil
	}
	result := make([]uint32, 0, len(empty.Cache))
	for u := range empty.Cache {
		result = append(result, u)
	}
	return result
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

func (p *neighborTable) Len() int {
	p.l.RLock()
	defer p.l.RUnlock()
	return len(p.cache)
}

func (p *neighborTable) List() []uint32 {
	p.l.RLock()
	defer p.l.RUnlock()
	result := make([]uint32, 0, len(p.cache))
	for id := range p.cache {
		result = append(result, id)
	}
	return result
}

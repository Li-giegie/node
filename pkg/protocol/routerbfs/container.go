package routerbfs

import (
	"github.com/Li-giegie/node/pkg/conn"
	"sync"
)

func newFullNodeContainer() *fullNodeContainer {
	return &fullNodeContainer{
		cache: make(map[uint32]map[uint32]int64),
	}
}

// fullNodeContainer 所有节点容器
type fullNodeContainer struct {
	// root-node -> sub-node -> node-unixNao
	cache map[uint32]map[uint32]int64
	sync.RWMutex
}

func (tab *fullNodeContainer) RootExist(id uint32) bool {
	tab.RLock()
	_, ok := tab.cache[id]
	tab.RUnlock()
	return ok
}

func (tab *fullNodeContainer) GetSubNodes(id uint32) []SubInfo {
	tab.RLock()
	defer tab.RUnlock()
	subTab, rootExist := tab.cache[id]
	if !rootExist {
		return nil
	}
	result := make([]SubInfo, 0, len(subTab))
	for u, i := range subTab {
		result = append(result, SubInfo{Id: u, UnixNao: i})
	}
	return result
}

func (tab *fullNodeContainer) Range(callback func(rootId, subId uint32, unixNano int64) bool) {
	tab.RLock()
	defer tab.RUnlock()
	for u, m := range tab.cache {
		for u2, i := range m {
			if !callback(u, u2, i) {
				return
			}
		}
	}
}

func (tab *fullNodeContainer) RangeWithRootNode(rootId uint32, callback func(subTab map[uint32]int64)) {
	tab.RLock()
	defer tab.RUnlock()
	subTab, rootExist := tab.cache[rootId]
	if !rootExist {
		return
	}
	callback(subTab)
	return
}

func (tab *fullNodeContainer) GetNodeUnixNano(rootId, subId uint32) (int64, bool) {
	tab.RLock()
	defer tab.RUnlock()
	subTab := tab.cache[rootId]
	unixNano, ok := subTab[subId]
	return unixNano, ok
}

func (tab *fullNodeContainer) Add(rootId uint32, subId uint32, subUnixNao int64) bool {
	tab.Lock()
	defer tab.Unlock()
	subTab, ok := tab.cache[rootId]
	if !ok {
		tab.cache[rootId] = map[uint32]int64{subId: subUnixNao}
		return true
	}
	if unixNano, ok := subTab[subId]; ok {
		if unixNano < subUnixNao {
			subTab[subId] = subUnixNao
		}
		return false
	}
	subTab[subId] = subUnixNao
	return true
}

func (tab *fullNodeContainer) Get(rootId uint32, subId uint32) (int64, bool) {
	tab.RLock()
	defer tab.RUnlock()
	subTab, ok := tab.cache[rootId]
	if !ok {
		return 0, false
	}
	unixNano, ok := subTab[subId]
	if !ok {
		return 0, false
	}
	return unixNano, true
}

func (tab *fullNodeContainer) Remove(rootId uint32, subId uint32, subUnixNao int64) bool {
	tab.Lock()
	defer tab.Unlock()
	rootNode, ok := tab.cache[rootId]
	if !ok {
		return false
	}
	unixNao, ok := rootNode[subId]
	if !ok || unixNao > subUnixNao {
		return false
	}
	// 删除父节点表中的子节点
	delete(rootNode, subId)
	if len(rootNode) == 0 {
		delete(tab.cache, rootId)
	}
	// 查询根节点表中子节点表数据
	subTab, ok := tab.cache[subId]
	if !ok {
		return true
	}
	// 子节点表的子节点不包含根节点，这个节点不是协议节点
	if _, ok = subTab[rootId]; !ok {
		return true
	}
	// 删除子节点为协议节点的关联节点
	for subTabId, _ := range subTab {
		rootNode, ok = tab.cache[subTabId]
		if ok {
			delete(rootNode, subId)
			if len(rootNode) == 0 {
				delete(tab.cache, subTabId)
			}
		}
	}
	delete(tab.cache, subId)
	return true
}

func (tab *fullNodeContainer) GetAllNodeInfo() []NodeInfo {
	tab.RLock()
	defer tab.RUnlock()
	res := make([]NodeInfo, 0, len(tab.cache))
	for rootId, info := range tab.cache {
		subInfo := make([]SubInfo, 0, len(info))
		for id, unixNao := range info {
			subInfo = append(subInfo, SubInfo{
				Id:      id,
				UnixNao: unixNao,
			})
		}
		res = append(res, NodeInfo{
			RootId: rootId,
			SubIds: subInfo,
		})
	}
	return res
}

func newProtoNodeContainer() *protoNodeContainer {
	return &protoNodeContainer{
		cache: make(map[uint32]conn.Conn),
	}
}

type protoNodeContainer struct {
	cache map[uint32]conn.Conn
	l     sync.RWMutex
}

func (p *protoNodeContainer) Add(id uint32, conn conn.Conn) {
	p.l.Lock()
	p.cache[id] = conn
	p.l.Unlock()
}

func (p *protoNodeContainer) Get(id uint32) (conn.Conn, bool) {
	p.l.RLock()
	c, ok := p.cache[id]
	p.l.RUnlock()
	return c, ok
}

func (p *protoNodeContainer) Remove(id uint32) {
	p.l.Lock()
	delete(p.cache, id)
	p.l.Unlock()
}

func (p *protoNodeContainer) Range(callback func(id uint32, conn conn.Conn) bool) {
	p.l.RLock()
	defer p.l.RUnlock()
	for u, c := range p.cache {
		if !callback(u, c) {
			return
		}
	}
}

func (p *protoNodeContainer) Len() int {
	p.l.RLock()
	l := len(p.cache)
	p.l.RUnlock()
	return l
}

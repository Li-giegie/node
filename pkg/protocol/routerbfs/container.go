package routerbfs

import (
	"github.com/Li-giegie/node/pkg/conn"
	"sync"
	"time"
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

func (tab *fullNodeContainer) Find(rootId uint32, callback func(subTab map[uint32]int64) interface{}) interface{} {
	tab.RLock()
	defer tab.RUnlock()
	subTab, rootExist := tab.cache[rootId]
	if !rootExist {
		return nil
	}
	return callback(subTab)
}

func (tab *fullNodeContainer) Range(callback func(rootId, subId uint32, unixNano int64)) {
	tab.RLock()
	defer tab.RUnlock()
	for u, m := range tab.cache {
		for u2, i := range m {
			callback(u, u2, i)
		}
	}
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
	if u, ok := subTab[subId]; ok && u > subUnixNao {
		return false
	}
	subTab[subId] = subUnixNao
	return true
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
	delete(rootNode, subId)
	if len(rootNode) == 0 {
		delete(tab.cache, rootId)
	}
	subTab, ok := tab.cache[subId]
	if !ok {
		return true
	}
	for subTabId, _ := range subTab {
		subTab, ok = tab.cache[subTabId]
		if ok {
			delete(subTab, subId)
			if len(subTab) == 0 {
				delete(tab.cache, subTabId)
			}
		}
	}
	delete(tab.cache, subId)
	return true
}

func (tab *fullNodeContainer) GetAllNodeInfo() []*NodeInfo {
	tab.RLock()
	defer tab.RUnlock()
	res := make([]*NodeInfo, 0, len(tab.cache))
	for rootId, info := range tab.cache {
		subInfo := make([]*SubInfo, 0, len(info))
		for id, unixNao := range info {
			subInfo = append(subInfo, &SubInfo{
				Id:      id,
				UnixNao: unixNao,
			})
		}
		res = append(res, &NodeInfo{
			RootNodeId:  rootId,
			SubNodeInfo: subInfo,
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

func (p *protoNodeContainer) Remove(id uint32) {
	p.l.Lock()
	delete(p.cache, id)
	p.l.Unlock()
}

func (p *protoNodeContainer) Range(callback func(id uint32, conn conn.Conn)) {
	p.l.RLock()
	for u, conn := range p.cache {
		callback(u, conn)
	}
	p.l.RUnlock()
}

// existContainer 记录消息协议消息Id是否重复
type existContainer struct {
	// k -> unixNao
	cache        map[uint64]int64
	ClearTimeout time.Duration
	sync.RWMutex
}

func newExistContainer(ClearTimeout time.Duration) *existContainer {
	return &existContainer{
		cache:        make(map[uint64]int64),
		RWMutex:      sync.RWMutex{},
		ClearTimeout: ClearTimeout,
	}
}

// Clean 清理value（unixNao）与当前时间差大于timeout的key
func (m *existContainer) Clean() {
	var timeoutKey []uint64
	m.RLock()
	for u, i := range m.cache {
		if time.Duration(time.Now().UnixNano()-i) >= m.ClearTimeout {
			timeoutKey = append(timeoutKey, u)
		}
	}
	m.RUnlock()
	if len(timeoutKey) > 0 {
		m.Lock()
		for _, u := range timeoutKey {
			delete(m.cache, u)
		}
		m.Unlock()
	}
}

func (m *existContainer) ExistOrStore(nodeId uint32, msgId uint32, unixNano int64) (exist bool) {
	m.Lock()
	defer m.Unlock()
	key := uint64(nodeId)<<32 | uint64(msgId)
	if l := len(m.cache); l > 0 && l%100 == 0 {
		m.Clean()
	}
	if _, exist = m.cache[key]; !exist {
		m.cache[key] = unixNano
	}
	return
}

func (m *existContainer) Remove(k uint32) {
	m.Lock()
	defer m.Unlock()
	for u, _ := range m.cache {
		if k == uint32(u>>32) {
			delete(m.cache, u)
		}
	}
}

func (m *existContainer) Get(k uint64) (int64, bool) {
	m.RLock()
	defer m.RUnlock()
	m2, ok := m.cache[k]
	return m2, ok
}

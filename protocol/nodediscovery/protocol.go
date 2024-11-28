package nodediscovery

import (
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	nodeNet "github.com/Li-giegie/node/net"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

func NewNodeDiscovery(protoType uint8, node Node, MaxRouteHop uint8, clearExistCacheTime time.Duration) NodeDiscoveryProtocol {
	p := &NodeDiscovery{
		node:                node,
		protoType:           protoType,
		MaxRouteHop:         MaxRouteHop,
		Router:              NewRouter(),
		nodeTab:             NewNodeTable(),
		existCache:          NewClearMap(),
		clearExistCacheTime: clearExistCacheTime,
	}
	node.AddOnConnect(p.OnConnect)
	node.AddOnCustomMessage(p.OnCustomMessage)
	node.AddOnForwardMessage(p.OnForwardMessage)
	node.AddOnClose(p.OnClose)
	unixNao := time.Now().UnixNano()
	for _, conn := range node.GetAllConn() {
		p.nodeTab.AddNode(node.Id(), conn.RemoteId(), unixNao)
	}
	return p
}

type NodeDiscovery struct {
	protoType           uint8
	idCounter           uint32
	counter             uint32
	MaxRouteHop         uint8
	clearExistCacheTime time.Duration
	node                Node
	nodeTab             *NodeTable
	existCache          *ClearMap
	*Router
}

func (p *NodeDiscovery) OnConnect(conn iface.Conn) {
	unixNao := time.Now().UnixNano()
	id := conn.RemoteId()
	p.nodeTab.AddNode(p.node.Id(), id, unixNao)
	p.broadcast(p.initProtoMsg(Action_AddNode, []*NodeInfo{
		{
			RootNodeId: p.node.Id(),
			SubNodeInfo: []*SubInfo{
				{
					Id:      id,
					UnixNao: unixNao,
				},
			},
		},
	}), 0, id)
	// 如果是桥接类型节点则发送本地所有节点和本地路由
	if conn.NodeType() == uint8(node.NodeType_Bridge) {
		p.existCache.Remove(id)
		p.WriteMsg(conn, 0, p.initProtoMsg(Action_AddNode, p.nodeTab.GetAllNodeInfo()).Encode())
	}
}

func (p *NodeDiscovery) OnCustomMessage(ctx iface.Context) {
	if ctx.Type() != p.protoType {
		return
	}
	p.counter++
	if p.counter%100 == 0 {
		go p.existCache.Clean(p.clearExistCacheTime)
	}
	if ctx.Hop() == 255 || ctx.Hop() >= p.MaxRouteHop && p.MaxRouteHop > 0 {
		return
	}
	var msg ProtoMsg
	if err := msg.Decode(ctx.Data()); err != nil {
		return
	}
	if exist := p.existCache.Add(uint64(msg.SrcId)<<32|uint64(msg.Id), time.Now().UnixNano()); exist {
		return
	}
	switch msg.Action {
	case Action_AddNode:
		for _, info := range msg.NInfo {
			for _, subInfo := range info.SubNodeInfo {
				p.nodeTab.AddNode(info.RootNodeId, subInfo.Id, subInfo.UnixNao)
			}
		}
	case Action_RemoveNode:
		for _, info := range msg.NInfo {
			for _, subInfo := range info.SubNodeInfo {
				p.existCache.Remove(subInfo.Id)
				p.nodeTab.RemoveNode(info.RootNodeId, subInfo.Id, subInfo.UnixNao)
				p.Router.RemoveRouteWithDst(subInfo.Id)
				p.Router.RemoveRouteWithVia(subInfo.Id)
				p.Router.RemoveRouteWithPath(subInfo.Id)
			}
		}
	default:
		return
	}
	p.broadcast(&msg, ctx.Hop(), ctx.SrcId(), msg.SrcId)
}

func (p *NodeDiscovery) OnClose(conn iface.Conn, err error) {
	id := conn.RemoteId()
	unixNao := time.Now().UnixNano()
	p.existCache.Remove(id)
	p.nodeTab.RemoveNode(p.node.Id(), id, unixNao)
	p.Router.RemoveRouteWithDst(id)
	p.Router.RemoveRouteWithVia(id)
	p.Router.RemoveRouteWithPath(id)
	p.broadcast(p.initProtoMsg(Action_RemoveNode, []*NodeInfo{
		{
			RootNodeId: p.node.Id(),
			SubNodeInfo: []*SubInfo{
				{
					Id:      id,
					UnixNao: unixNao,
				},
			},
		},
	}), 0)
}

func (p *NodeDiscovery) OnForwardMessage(ctx iface.Context) {
	if ctx.Hop() >= 254 || ctx.Hop() > p.MaxRouteHop && p.MaxRouteHop > 0 {
		log.Println("OnForwardMessage", ctx.Hop())
		return
	}
	empty, exist := p.GetRoute(ctx.DestId())
	if !exist {
		_ = ctx.ReplyError(nodeNet.ErrNodeNotExist, nil)
		return
	}
	conn, exist := p.node.GetConn(empty.via)
	if !exist {
		_ = ctx.ReplyError(nodeNet.ErrNodeNotExist, nil)
		return
	}
	_, _ = conn.WriteMsg(&message.Message{
		Type:   ctx.Type(),
		Hop:    ctx.Hop(),
		Id:     ctx.Id(),
		SrcId:  ctx.SrcId(),
		DestId: ctx.DestId(),
		Data:   ctx.Data(),
	})
}

func (p *NodeDiscovery) initProtoMsg(action uint8, info []*NodeInfo) *ProtoMsg {
	m := ProtoMsg{
		Id:     atomic.AddUint32(&p.idCounter, 1),
		SrcId:  p.node.Id(),
		Action: action,
		NInfo:  info,
	}
	p.existCache.Add(uint64(p.node.Id())<<32|uint64(m.Id), time.Now().UnixNano())
	return &m
}

func (p *NodeDiscovery) broadcast(m *ProtoMsg, hop uint8, filterId ...uint32) {
	var has bool
	data := m.Encode()
	conns := p.node.GetAllConn()
	for _, conn := range conns {
		if conn.NodeType() == uint8(node.NodeType_Base) {
			continue
		}
		has = false
		for _, id := range filterId {
			if id == conn.RemoteId() {
				has = true
				break
			}
		}
		if has {
			continue
		}
		p.WriteMsg(conn, hop, data)
	}
}

func (p *NodeDiscovery) WriteMsg(conn iface.Conn, hop uint8, data []byte) {
	_, _ = conn.WriteMsg(&message.Message{
		Type:   p.protoType,
		Hop:    hop,
		SrcId:  conn.LocalId(),
		DestId: conn.RemoteId(),
		Data:   data,
	})
}

type bfsValue struct {
	val     uint32
	unixNao int64
	paths   []uint32
}

func (p *NodeDiscovery) BFS(target uint32) ([]uint32, int64, bool) {
	p.nodeTab.l.RLock()
	defer p.nodeTab.l.RUnlock()
	id := p.node.Id()
	queue := []*bfsValue{{val: id, paths: []uint32{id}}}
	exist := map[uint32]struct{}{id: {}}
	for len(queue) > 0 {
		current := queue[0]
		if current.val == target {
			return current.paths, current.unixNao, true
		}
		queue = queue[1:]
		m2, ok := p.nodeTab.Cache[current.val]
		if !ok {
			continue
		}
		if unixNao, ok := m2[target]; ok {
			return append(current.paths, target), unixNao, true
		}
		for u, unixNao := range p.nodeTab.Cache[current.val] {
			if _, ok = exist[u]; !ok {
				queue = append(queue, &bfsValue{val: u, unixNao: unixNao, paths: append(current.paths, u)})
				exist[u] = struct{}{}
			}
		}
	}
	return nil, 0, false
}

func (p *NodeDiscovery) GetRoute(dst uint32) (*RouteEmpty, bool) {
	if dst == p.node.Id() {
		return nil, false
	}
	empty, ok := p.Router.GetRoute(dst)
	if ok {
		return empty, true
	}
	fullPath, unixNao, ok := p.BFS(dst)
	if ok {
		hop := len(fullPath) - 1
		if hop <= 0 || hop > 255 || hop > int(p.MaxRouteHop) {
			return nil, false
		}
		p.AddRouteWithFullPath(dst, fullPath[1], uint8(hop), unixNao, fullPath)
		return &RouteEmpty{
			dst:      dst,
			via:      fullPath[1],
			hop:      uint8(hop),
			duration: time.Duration(unixNao),
			fullPath: fullPath,
		}, true
	}
	return nil, false
}

func (p *NodeDiscovery) RangeNode(callback func(root uint32, sub []*SubInfo)) {
	for _, info := range p.nodeTab.GetAllNodeInfo() {
		callback(info.RootNodeId, info.SubNodeInfo)
	}
}

func NewNodeTable() *NodeTable {
	return &NodeTable{
		Cache: make(map[uint32]map[uint32]int64),
		l:     sync.RWMutex{},
	}
}

type NodeTable struct {
	// root-node -> sub-node -> node-unixNao
	Cache map[uint32]map[uint32]int64
	l     sync.RWMutex
}

func (tab *NodeTable) AddNode(rootId uint32, subId uint32, subUnixNao int64) {
	tab.l.Lock()
	defer tab.l.Unlock()
	subTab, ok := tab.Cache[rootId]
	if !ok {
		tab.Cache[rootId] = map[uint32]int64{subId: subUnixNao}
		return
	}
	if u, ok := subTab[subId]; ok && u > subUnixNao {
		return
	}
	subTab[subId] = subUnixNao
}

// RemoveNode 移除根节点中的节点
func (tab *NodeTable) RemoveNode(rootId uint32, subId uint32, subUnixNao int64) {
	tab.l.Lock()
	defer tab.l.Unlock()
	rootNode, ok := tab.Cache[rootId]
	if !ok {
		return
	}
	unixNao, ok := rootNode[subId]
	if !ok || unixNao > subUnixNao {
		return
	}
	delete(rootNode, subId)
	if len(rootNode) == 0 {
		delete(tab.Cache, rootId)
	}
	subTab, ok := tab.Cache[subId]
	if !ok {
		return
	}
	for subTabId, _ := range subTab {
		subTab, ok = tab.Cache[subTabId]
		if ok {
			delete(subTab, subId)
			if len(subTab) == 0 {
				delete(tab.Cache, subTabId)
			}
		}
	}
	delete(tab.Cache, subId)
}

func (tab *NodeTable) GetAllNodeInfo() []*NodeInfo {
	tab.l.RLock()
	defer tab.l.RUnlock()
	res := make([]*NodeInfo, 0, len(tab.Cache))
	for rootId, info := range tab.Cache {
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

type ClearMap struct {
	// k -> unixNao
	cache map[uint64]int64
	sync.RWMutex
}

func NewClearMap() *ClearMap {
	return &ClearMap{
		cache:   make(map[uint64]int64),
		RWMutex: sync.RWMutex{},
	}
}

// Clean 清理value（unixNao）与当前时间差大于timeout的key
func (m *ClearMap) Clean(timeout time.Duration) {
	var timeoutKey []uint64
	m.RLock()
	for u, i := range m.cache {
		if time.Duration(time.Now().UnixNano()-i) >= timeout {
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

func (m *ClearMap) Add(key uint64, unixNao int64) (isAdd bool) {
	m.Lock()
	defer m.Unlock()
	if _, isAdd = m.cache[key]; !isAdd {
		m.cache[key] = unixNao
	}
	return
}

func (m *ClearMap) Remove(k uint32) {
	m.Lock()
	defer m.Unlock()
	for u, _ := range m.cache {
		if k == uint32(u>>32) {
			delete(m.cache, u)
		}
	}
}

func (m *ClearMap) Get(k uint64) (int64, bool) {
	m.RLock()
	defer m.RUnlock()
	m2, ok := m.cache[k]
	return m2, ok
}

package routerbfs

import (
	"context"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/handler"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"github.com/Li-giegie/node/pkg/router"
	"time"
)

type Node interface {
	NodeId() uint32
	GetAllConn() []conn.Conn
	LenConn() (n int)
	GetConn(id uint32) (conn.Conn, bool)
	GetRouter() router.Router
	RangeConn(f func(conn conn.Conn) bool)
}

func NewRouterBFS(protoType uint8, node Node, MaxRouteHop uint8) *RouterBFS {
	p := &RouterBFS{
		node:        node,
		protoType:   protoType,
		maxRouteHop: MaxRouteHop,
		FullNode:    newFullNodeContainer(),
		ProtoNode:   newProtoNodeContainer(),
	}
	node.GetRouter().ReroutingHandleFunc(p.CalcRoute)
	return p
}

type RouterBFS struct {
	handler.Empty
	protoType   uint8               //协议类型
	maxRouteHop uint8               //最大路由跳数
	node        Node                // 当前节点
	FullNode    *fullNodeContainer  //全部节点
	ProtoNode   *protoNodeContainer //直连的协议节点
	once        bool                // 是否第一次OnConnect
}

func (p *RouterBFS) ProtocolType() uint8 {
	return p.protoType
}

func (p *RouterBFS) OnConnect(conn conn.Conn) {
	// 如果once为true，则协议询问过全部节点至少一次其他是否开启协议，主要用于节点在开启协议前已经有桥接节点，但并未开启协议，之后又开启了协议
	if p.once {
		go p.onConnect(conn)
		return
	}
	// 初始化
	p.once = true
	conns := p.node.GetAllConn()
	for i := 0; i < len(conns); i++ {
		go p.onConnect(conns[i])
	}
}

func (p *RouterBFS) onConnect(conn conn.Conn) {
	subId := conn.RemoteId()
	unixNao := time.Now().UnixNano()
	currId := p.node.NodeId()
	p.FullNode.Add(currId, subId, unixNao)
	p.node.GetRouter().RemoveRouteWithPath(subId)
	m := ProtoMsg{
		Action: Action_AddNode,
		Paths:  []uint32{currId, subId},
		Nodes: []NodeInfo{
			{
				RootId: currId,
				SubIds: []SubInfo{{Id: subId, UnixNao: unixNao}},
			},
		},
	}
	// 广播通知协议节点，当前节点有新节点上线
	p.broadcastV2(&m, 0)
	// 查询对端节点是否开启当前协议
	m.Action = Action_QueryProtocol
	m.Paths = []uint32{currId}
	_ = conn.SendType(p.protoType, m.Encode())
}

func (p *RouterBFS) OnMessage(r responsewriter.ResponseWriter, msg *message.Message) {
	// 防环最后一道防线
	if msg.Hop >= 254 || msg.Hop >= p.maxRouteHop && p.maxRouteHop > 0 {
		return
	}
	proto, err := decodeProtoMsg(msg.Data)
	if err != nil {
		return
	}
	currId := p.node.NodeId()
	// 防环，协议消息出现当前节点Id认为该Id已处理过该消息，正常情况不会触发
	for _, path := range proto.Paths {
		if currId == path {
			return
		}
	}
	// 当前节点添加到已处理路径中
	proto.Paths = append(proto.Paths, currId)
	switch proto.Action {
	case Action_QueryProtocol: //走单播
		proto.Action = Action_ReplyProtocol
		proto.Paths = []uint32{currId}
		proto.Nodes = p.FullNode.GetAllNodeInfo()
		proto.SyncInfo = nil
		_ = r.GetConn().SendTypeTo(p.protoType, msg.SrcId, proto.Encode())
		return
	case Action_ReplyProtocol: // 单播接收，广播转发，添加协议节点，通知直连的协议节点
		p.ProtoNode.Add(msg.SrcId, r.GetConn())
		p.addNodes(proto.Nodes)
		proto.Action = Action_AddNode
	case Action_AddNode: // 添加节点
		p.addNodes(proto.Nodes)
	case Action_RemoveNode: //删除节点
		p.removeNodes(proto.Nodes)
	case Action_SyncHash: //同步节点哈希
		if proto.SyncInfo == nil || proto.SyncInfo.ValidityUnixNano < time.Now().UnixNano() {
			return
		}
		hash := uint64(0)
		nodeNum := uint32(0)
		p.FullNode.RangeWithRootNode(msg.SrcId, func(subTab map[uint32]int64) {
			nodeNum = uint32(len(subTab))
			for u, _ := range subTab {
				hash += uint64(u)
			}
		})
		if nodeNum != proto.SyncInfo.NodeNum || hash != proto.SyncInfo.Hash {
			proto.Action = Action_SyncQueryNode
			proto.SyncInfo.Hash = hash
			proto.SyncInfo.NodeNum = nodeNum
			proto.Paths = []uint32{currId}
			_ = r.GetConn().SendTypeTo(p.protoType, msg.SrcId, proto.Encode())
		}
		return
	case Action_SyncQueryNode: //单播
		if proto.SyncInfo == nil {
			return
		}
		hash := uint64(0)
		syncNode := SyncNode{RootId: currId}
		p.FullNode.RangeWithRootNode(currId, func(subTab map[uint32]int64) {
			syncNode.SubIds = make([]uint32, 0, len(subTab))
			for u, _ := range subTab {
				hash += uint64(u)
				syncNode.SubIds = append(syncNode.SubIds, u)
			}
		})
		if proto.SyncInfo.NodeNum == uint32(len(syncNode.SubIds)) && hash == proto.SyncInfo.Hash {
			return
		}
		proto.Action = Action_SyncReplyNode
		proto.SyncInfo.Hash = hash
		proto.SyncInfo.NodeNum = uint32(len(syncNode.SubIds))
		proto.SyncInfo.SyncNode = &syncNode
		proto.Paths = []uint32{currId}
		proto.SyncInfo.ValidityUnixNano = time.Now().UnixNano()
		_ = r.GetConn().SendTypeTo(p.protoType, msg.SrcId, proto.Encode())
		return
	case Action_SyncReplyNode: //单播接收，广播发出
		if proto.SyncInfo == nil || proto.SyncInfo.SyncNode == nil {
			return
		}
		isForward := false
		syncTab := make(map[uint32]struct{}, len(proto.SyncInfo.SubIds))
		for _, id := range proto.SyncInfo.SubIds {
			syncTab[id] = struct{}{}
		}
		var removeList []uint32
		// 删除
		p.FullNode.RangeWithRootNode(proto.SyncInfo.RootId, func(subTab map[uint32]int64) {
			for u, i := range subTab {
				if proto.SyncInfo.ValidityUnixNano > i {
					if _, ok := syncTab[u]; !ok {
						removeList = append(removeList, u)
					}
				}
			}
		})
		// 更新删除
		for _, id := range removeList {
			if p.FullNode.Remove(proto.SyncInfo.RootId, id, proto.SyncInfo.ValidityUnixNano) {
				p.removeRoute(id)
				isForward = true
			}
		}
		// 添加
		for _, id := range proto.SyncInfo.SubIds {
			if p.FullNode.Add(proto.SyncInfo.RootId, id, proto.SyncInfo.ValidityUnixNano) {
				isForward = true
				p.addRoute(id, proto.SyncInfo.ValidityUnixNano)
			}
		}
		if !isForward {
			return
		}
	}
	p.broadcastV2(proto, msg.Hop)
}

func (p *RouterBFS) OnClose(conn conn.Conn, err error) {
	id := conn.RemoteId()
	currId := p.node.NodeId()
	unixNano := time.Now().UnixNano()
	if p.FullNode.Remove(currId, id, unixNano) {
		p.ProtoNode.Remove(id)
		p.FullNode.Remove(currId, id, unixNano)
		p.removeRoute(id)
		m := ProtoMsg{
			Action: Action_RemoveNode,
			Paths:  []uint32{currId},
			Nodes:  []NodeInfo{{RootId: currId, SubIds: []SubInfo{{id, unixNano}}}},
		}
		p.broadcastV2(&m, 0)
	}
}

func (p *RouterBFS) addNodes(infos []NodeInfo) {
	var success []NodeInfo
	currId := p.node.NodeId()
	for _, info := range infos {
		if info.RootId == currId {
			continue
		}
		item := NodeInfo{RootId: info.RootId}
		for _, subInfo := range info.SubIds {
			if p.FullNode.Add(info.RootId, subInfo.Id, subInfo.UnixNao) {
				item.SubIds = append(item.SubIds, subInfo)
			}
		}
		if len(item.SubIds) > 0 {
			success = append(success, item)
		}
	}
	p.addRoutes(success)
}

func (p *RouterBFS) removeNodes(infos []NodeInfo) {
	var success []NodeInfo
	currId := p.node.NodeId()
	for _, info := range infos {
		if info.RootId == currId {
			continue
		}
		item := NodeInfo{RootId: info.RootId}
		for _, subInfo := range info.SubIds {
			if p.FullNode.Remove(info.RootId, subInfo.Id, subInfo.UnixNao) {
				item.SubIds = append(item.SubIds, subInfo)
			}
		}
		if len(item.SubIds) > 0 {
			success = append(success, item)
		}
	}
	p.removeRoutes(success)
}

func (p *RouterBFS) addRoutes(infos []NodeInfo) {
	route := p.node.GetRouter()
	curId := p.node.NodeId()
	unixNano := time.Now().UnixNano()
	for _, info := range infos {
		if info.RootId == curId {
			continue
		}
		empty, ok := p.addRoute(info.RootId, unixNano)
		if ok {
			pl := len(empty.Paths)
			for _, subInfo := range info.SubIds {
				if subInfo.Id != curId {
					if _, ok = p.node.GetConn(subInfo.Id); !ok {
						subPaths := make([]uint32, len(empty.Paths)+1)
						copy(subPaths, empty.Paths)
						subPaths[pl] = subInfo.Id
						route.AddRoute(subInfo.Id, empty.Via, uint8(pl), unixNano, subPaths)
					}
				}
			}
		}
	}
}

func (p *RouterBFS) addRoute(id uint32, unixNano int64) (*router.RouteEmpty, bool) {
	if id == p.node.NodeId() {
		return nil, false
	}
	if _, ok := p.node.GetConn(id); ok {
		return &router.RouteEmpty{Dst: id, Via: id, Hop: 1, Paths: []uint32{p.node.NodeId(), id}}, true
	}
	empty, ok := p.CalcRoute(id)
	if !ok {
		return nil, false
	}
	p.node.GetRouter().AddRoute(empty.Dst, empty.Via, empty.Hop, unixNano, empty.Paths)
	return empty, true
}

func (p *RouterBFS) removeRoutes(infos []NodeInfo) {
	curId := p.node.NodeId()
	route := p.node.GetRouter()
	var removeRoutes []uint32
	for _, info := range infos {
		for _, subInfo := range info.SubIds {
			if subInfo.Id == curId {
				continue
			}
			removeRoutes = append(removeRoutes, subInfo.Id)
			removeRoutes = append(removeRoutes, route.GetRouteDstWithPath(subInfo.Id)...)
			route.RemoveRouteWithPath(subInfo.Id)
		}
	}
	unixNano := time.Now().UnixNano()
	for _, removeRoute := range removeRoutes {
		if removeRoute == curId {
			continue
		}
		p.addRoute(removeRoute, unixNano)
	}
}

func (p *RouterBFS) removeRoute(id uint32) {
	curId := p.node.NodeId()
	if id == curId {
		return
	}
	unixNano := time.Now().UnixNano()
	route := p.node.GetRouter()
	removeRoutes := route.GetRouteDstWithPath(id)
	route.RemoveRouteWithPath(id)
	p.addRoute(id, unixNano)
	for _, removeRoute := range removeRoutes {
		if removeRoute == curId {
			continue
		}
		p.addRoute(removeRoute, unixNano)
	}
}

func (p *RouterBFS) broadcastV2(msg *ProtoMsg, hop uint8) {
	p.ProtoNode.Len()
	num := p.node.LenConn()
	if num == 0 {
		return
	}
	var filterIds = make(map[uint32]struct{}, len(msg.Paths))
	for _, path := range msg.Paths {
		filterIds[path] = struct{}{}
	}
	var conns = make([]conn.Conn, 0, num)
	var connIds = make([]uint32, 0, num)
	p.ProtoNode.Range(func(id uint32, conn conn.Conn) bool {
		if _, ok := filterIds[id]; ok {
			return true
		}
		connIds = append(connIds, id)
		conns = append(conns, conn)
		return true
	})
	if len(conns) == 0 {
		return
	}
	if len(conns) == 1 {
		_ = conns[0].SendMessage(&message.Message{
			Type:   p.protoType,
			Hop:    hop,
			SrcId:  conns[0].LocalId(),
			DestId: conns[0].RemoteId(),
			Data:   msg.Encode(),
		})
		return
	}
	l := len(msg.Paths)
	for i, c := range conns {
		msg.Paths = append(msg.Paths, connIds[:i]...)
		msg.Paths = append(msg.Paths, connIds[i+1:]...)
		m := &message.Message{
			Type:   p.protoType,
			Hop:    hop,
			SrcId:  c.LocalId(),
			DestId: c.RemoteId(),
			Data:   msg.Encode(),
		}
		_ = c.SendMessage(m)
		msg.Paths = msg.Paths[:l]
	}
}

type bfsResult struct {
	node  uint32
	paths []uint32
}

// BFSSearch 搜索起点到终端的路径，src起点，dst终点，maxDeep最大深度，返回起点到终点的全部路径，bool是否存在
func (p *RouterBFS) BFSSearch(src, dst uint32, maxDeep uint8) (result []uint32) {
	if src == dst {
		return nil
	}
	queue := make([]*bfsResult, 1, 10)
	queue[0] = &bfsResult{node: src, paths: []uint32{src}}
	existTab := map[uint32]struct{}{src: {}}
	var subId uint32
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if len(current.paths) >= int(maxDeep) {
			return nil
		}
		p.FullNode.RangeWithRootNode(current.node, func(subTab map[uint32]int64) {
			var ok bool
			if _, ok = subTab[dst]; ok {
				result = append(current.paths, dst)
				return
			}
			for subId, _ = range subTab {
				if _, ok = existTab[subId]; ok {
					continue
				}
				existTab[subId] = struct{}{}
				queue = append(queue, &bfsResult{
					node:  subId,
					paths: append(current.paths, subId)},
				)
			}
			return
		})
		if len(result) > 0 {
			return result
		}
	}
	return nil
}

func (p *RouterBFS) CalcRoute(dst uint32) (*router.RouteEmpty, bool) {
	paths := p.BFSSearch(p.node.NodeId(), dst, p.maxRouteHop)
	if len(paths) < 2 {
		return nil, false
	}
	empty := &router.RouteEmpty{
		Dst:   dst,
		Via:   paths[1],
		Hop:   uint8(len(paths) - 1),
		Paths: paths,
	}
	return empty, true
}

func (p *RouterBFS) StartNodeSync(ctx context.Context, timeout time.Duration) {
	go func() {
		currId := p.node.NodeId()
		ticker := time.NewTicker(timeout)
		defer ticker.Stop()
		go func() {
			<-ctx.Done()
			ticker.Stop()
		}()
		m := ProtoMsg{
			Action:   Action_SyncHash,
			Paths:    []uint32{currId},
			SyncInfo: &SyncInfo{},
		}
		for t := range ticker.C {
			m.SyncInfo.Hash = 0
			m.SyncInfo.NodeNum = 0
			p.FullNode.RangeWithRootNode(currId, func(subTab map[uint32]int64) {
				m.SyncInfo.NodeNum = uint32(len(subTab))
				for id := range subTab {
					m.SyncInfo.Hash += uint64(id)
				}
			})
			if m.SyncInfo.NodeNum > 0 {
				m.SyncInfo.ValidityUnixNano = t.UnixNano() + int64(timeout)
				p.broadcastV2(&m, 0)
			}
		}
	}()
}

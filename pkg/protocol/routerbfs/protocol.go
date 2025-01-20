package routerbfs

import (
	"context"
	"encoding/json"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/errors"
	"github.com/Li-giegie/node/pkg/handler"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"github.com/Li-giegie/node/pkg/router"
	"github.com/sirupsen/logrus"
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
		node:          node,
		protoType:     protoType,
		maxRouteHop:   MaxRouteHop,
		nodeTable:     newNodeTable(),
		neighborTable: newNeighborTable(),
		checkTable:    newCheckTable(5),
		init:          true,
		nodeId:        node.NodeId(),
	}
	node.GetRouter().ReroutingHandleFunc(p.CalcRoute)
	return p
}

type RouterBFS struct {
	handler.Empty
	protoType      uint8 //协议类型
	maxRouteHop    uint8 //最大路由跳数
	init           bool  // 是否第一次OnConnect
	node           Node  // 当前节点
	nodeId         uint32
	*nodeTable     //全部节点
	*neighborTable //直连的协议节点
	*checkTable    //检查消息时间戳
}

func (p *RouterBFS) ProtocolType() uint8 {
	return p.protoType
}

func (p *RouterBFS) OnConnect(c conn.Conn) {
	if p.init {
		p.init = false
		p.nodeTable.AddRoot(p.node.NodeId())
		p.node.RangeConn(func(c conn.Conn) bool {
			go p.onConnect(c)
			return true
		})
		return
	}
	go p.onConnect(c)
}

func (p *RouterBFS) onConnect(c conn.Conn) {
	subId := c.RemoteId()
	unixNao := time.Now().UnixNano()
	p.nodeTable.AddSub(p.nodeId, subId)
	logrus.Infoln("OnConnect", subId)
	// 广播通知协议节点，当前节点有新节点上线
	updateList := []*UpdateMsg{
		{
			Action: UpdateAction_AddSub,
			RootId: p.node.NodeId(),
			SubId:  subId,
		},
	}
	data, _ := json.Marshal(updateList)
	p.broadcast(0, &ProtoMsg{
		Action:   Action_Update,
		Paths:    []uint32{p.nodeId, subId},
		UnixNano: unixNao,
		SrcId:    p.nodeId,
		Data:     data,
	})
	// 查询对端节点是否开启当前协议
	if c.NodeType() == conn.NodeTypeServer {
		p.send(c, &ProtoMsg{
			Action:   Action_NeighborASK,
			UnixNano: unixNao,
			SrcId:    p.nodeId,
			Paths:    []uint32{p.nodeId},
		})
		logrus.Infoln("OnConnect NeighborASK", p.nodeId)
	}
}

func (p *RouterBFS) OnMessage(r responsewriter.ResponseWriter, msg *message.Message) {
	proto, err := p.validMessage(msg)
	if err != nil {
		if proto != nil {
			logrus.Errorln(err, proto.String())
		} else {
			logrus.Errorln(err)
		}
		return
	}
	logrus.Infoln("OnMessage", proto.String())
	unixNao := time.Now().UnixNano()
	// 当前节点添加到已处理路径中
	proto.Paths = append(proto.Paths, p.nodeId)
	switch proto.Action {
	case Action_NeighborASK: //走单播
		data, err := p.nodeTable.Encode()
		if err != nil {
			logrus.Errorln(err)
			return
		}
		p.send(r.GetConn(), &ProtoMsg{
			Action:   Action_NeighborACK,
			UnixNano: unixNao,
			SrcId:    p.nodeId,
			Paths:    []uint32{p.nodeId},
			Data:     data,
		})
	case Action_NeighborACK:
		var nodeList map[uint32]map[uint32]struct{}
		if err = json.Unmarshal(proto.Data, &nodeList); err != nil {
			logrus.Errorln("解码失败：", err)
			return
		}
		if !p.neighborTable.AddNeighbor(proto.SrcId, r.GetConn()) {
			logrus.Warningln("邻居已存在：", proto.SrcId)
			return
		}
		for rId, empty := range nodeList {
			if !p.nodeTable.AddRoot(rId) {
				delete(nodeList, rId)
				logrus.Warningln("添加节点失败节点存在", rId)
				return
			}
			for sId := range empty {
				if !p.nodeTable.AddSub(rId, sId) {
					delete(empty, sId)
				}
			}
		}
		for rId, empty := range nodeList {
			rInfo, ok := p.addRoute(rId)
			if !ok {
				delete(nodeList, rId)
				logrus.Warningln("添加路由失败已存在", rId)
				continue
			}
			route := p.node.GetRouter()
			for sId := range empty {
				if sId == p.nodeId {
					continue
				}
				if _, ok = p.node.GetConn(sId); ok {
					continue
				}
				paths := make([]uint32, len(rInfo.Paths)+1)
				copy(paths, rInfo.Paths)
				paths[len(rInfo.Paths)] = sId
				route.AddRoute(sId, rId, rInfo.Hop+1, time.Now().UnixNano(), paths)
			}
		}
		updateList := make([]*UpdateMsg, 0, len(nodeList))
		for u, empty := range nodeList {
			updateList = append(updateList, &UpdateMsg{
				Action: UpdateAction_AddRoot,
				RootId: u,
			})
			for sId := range empty {
				updateList = append(updateList, &UpdateMsg{
					Action: UpdateAction_AddSub,
					RootId: u,
					SubId:  sId,
				})
			}
		}
		data, _ := json.Marshal(updateList)
		p.broadcast(0, &ProtoMsg{
			Action:   Action_Update,
			UnixNano: unixNao,
			SrcId:    p.nodeId,
			Paths:    []uint32{p.nodeId, proto.SrcId},
			Data:     data,
		})
	case Action_Update:
		var updateList []*UpdateMsg
		if err = json.Unmarshal(proto.Data, &updateList); err != nil {
			logrus.Errorln("Action_Update,err:", err)
			return
		}
		successUpdates := make([]*UpdateMsg, 0, len(updateList))
		var success bool
		for _, update := range updateList {
			if update.RootId == p.nodeId {
				continue
			}
			switch update.Action {
			case UpdateAction_AddRoot:
				success = p.AddRoot(update.RootId)
			case UpdateAction_AddSub:
				success = p.AddSub(update.RootId, update.SubId)
			case UpdateAction_RemoveRoot:
				success = p.RemoveRoot(update.RootId)
			case UpdateAction_DeleteSub:
				success = p.DeleteSub(update.RootId, update.SubId)
			default:
				success = false
			}
			if success {
				successUpdates = append(successUpdates, update)
			}
		}
		for _, update := range successUpdates {
			switch update.Action {
			case UpdateAction_AddRoot:
				p.addRoute(update.RootId)
			case UpdateAction_AddSub:
				p.addRoute(update.SubId)
			case UpdateAction_RemoveRoot:
				p.removeRoute(update.RootId)
			case UpdateAction_DeleteSub:
				p.removeRoute(update.SubId)
			}
		}
		data, _ := json.Marshal(successUpdates)
		proto.Data = data
		p.broadcast(msg.Hop, proto)
	case Action_SyncHash:
		var syncHashs []*SyncMsg
		if err = json.Unmarshal(proto.Data, &syncHashs); err != nil {
			return
		}
		var needs []*SyncMsg
		for _, hash := range syncHashs {
			if hash.Id == p.nodeId {
				continue
			}
			result := p.getNodeHash(hash.Id)
			if !equalSyncHash(result, hash) {
				if result == nil {
					hash.Hash = 0
					hash.SubNodeNum = 0
				} else {
					hash.Hash = result.Hash
					hash.SubNodeNum = result.SubNodeNum
				}
				needs = append(needs, hash)
			}
		}
		if len(needs) > 0 {
			data, _ := json.Marshal(needs)
			p.send(r.GetConn(), &ProtoMsg{
				Action:   Action_SyncQueryNode,
				UnixNano: unixNao,
				SrcId:    p.nodeId,
				Paths:    []uint32{p.nodeId},
				Data:     data,
			})
		}
	case Action_SyncQueryNode: //请求同步，单播
		var needs []*SyncMsg
		if err = json.Unmarshal(proto.Data, &needs); err != nil {
			logrus.Errorln(err)
			return
		}
		updateList := make([]*UpdateMsg, 0, len(needs))
		for _, need := range needs {
			result := p.getNodeHash(need.Id)
			if result == nil {
				updateList = append(updateList, &UpdateMsg{
					Action: UpdateAction_RemoveRoot,
					RootId: need.Id,
				})
				continue
			}
			if !equalSyncHash(result, need) {
				updateList = append(updateList, &UpdateMsg{
					Action: UpdateAction_RemoveRoot,
					RootId: need.Id,
				})
				if result.SubNodeNum == 0 {
					continue
				}
				updateList = append(updateList, &UpdateMsg{
					Action: UpdateAction_AddRoot,
					RootId: need.Id,
				})
				for _, u := range p.nodeTable.GetSubNodes(need.Id) {
					updateList = append(updateList, &UpdateMsg{
						Action: UpdateAction_AddSub,
						RootId: need.Id,
						SubId:  u,
					})
				}
			}
		}
		if len(updateList) > 0 {
			data, _ := json.Marshal(updateList)
			p.send(r.GetConn(), &ProtoMsg{
				Action:   Action_Update,
				UnixNano: unixNao,
				SrcId:    p.nodeId,
				Paths:    []uint32{p.nodeId},
				Data:     data,
			})
		}
	}
}

func (p *RouterBFS) OnClose(conn conn.Conn, err error) {
	subId := conn.RemoteId()
	unixNano := time.Now().UnixNano()
	p.nodeTable.DeleteSub(p.nodeId, subId)
	updates := []*UpdateMsg{
		{
			Action: UpdateAction_DeleteSub,
			RootId: p.nodeId,
			SubId:  subId,
		},
	}
	if _, ok := p.neighborTable.GetNeighbor(subId); ok {
		p.neighborTable.DeleteNeighbor(subId)
		p.nodeTable.RemoveRoot(subId)
		updates = append(updates, &UpdateMsg{
			Action: UpdateAction_RemoveRoot,
			RootId: subId,
		})
	}
	data, _ := json.Marshal(updates)
	p.broadcast(0, &ProtoMsg{
		Action:   Action_Update,
		UnixNano: unixNano,
		SrcId:    p.nodeId,
		Paths:    []uint32{p.nodeId, subId},
		Data:     data,
	})
}

func (p *RouterBFS) validMessage(msg *message.Message) (*ProtoMsg, error) {
	if msg.Hop >= 254 || msg.Hop >= p.maxRouteHop && p.maxRouteHop > 0 {
		return nil, errors.New("hop overflow Loop messages")
	}
	proto, err := decodeProtoMsg(msg.Data)
	if err != nil {
		return nil, err
	}
	if err = proto.Valid(); err != nil {
		return proto, err
	}
	for _, path := range proto.Paths {
		if path == p.nodeId {
			return proto, errors.New("invalid path")
		}
	}
	if !p.Permit(proto.SrcId, proto.UnixNano) {
		return proto, errors.New("invalid unixNano")
	}
	return proto, nil
}

func (p *RouterBFS) addRoute(id uint32) (*router.RouteEmpty, bool) {
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
	p.node.GetRouter().AddRoute(empty.Dst, empty.Via, empty.Hop, time.Now().UnixNano(), empty.Paths)
	return empty, true
}

func (p *RouterBFS) removeRoute(id uint32) {
	curId := p.node.NodeId()
	if id == curId {
		return
	}
	route := p.node.GetRouter()
	removeRoutes := route.GetRouteDstWithPath(id)
	route.RemoveRouteWithPath(id)
	p.addRoute(id)
	for _, removeRoute := range removeRoutes {
		if removeRoute == curId {
			continue
		}
		p.addRoute(removeRoute)
	}
}

func (p *RouterBFS) send(conn conn.Conn, msg *ProtoMsg, debugPrefix ...string) {
	if len(debugPrefix) > 0 {
		logrus.Infoln("[Send]", debugPrefix[0], msg.String())
	} else {
		logrus.Infoln("[Send]", msg.String())
	}
	conn.SendType(p.protoType, msg.Encode())
}

func (p *RouterBFS) broadcast(hop uint8, msg *ProtoMsg) {
	p.LenNeighbor()
	num := p.node.LenConn()
	if num == 0 {
		return
	}
	var filterIds = make(map[uint32]struct{}, len(msg.Paths))
	for _, path := range msg.Paths {
		filterIds[path] = struct{}{}
	}
	var conns = make([]*Conn, 0, num)
	var connIds = make([]uint32, 0, num)
	p.RangeNeighbor(func(id uint32, conn *Conn) bool {
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
		p.RangeSubNode(current.node, func(empty map[uint32]struct{}) {
			var ok bool
			if _, ok = empty[dst]; ok {
				result = append(current.paths, dst)
				return
			}
			for subId = range empty {
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
		m := &ProtoMsg{
			Action: Action_SyncHash,
			SrcId:  currId,
			Paths:  []uint32{currId},
		}
		for t := range ticker.C {
			if p.neighborTable.LenNeighbor() == 0 {
				continue
			}
			syncList := make([]*SyncMsg, 0, 1+p.neighborTable.LenNeighbor())
			p.neighborTable.RangeNeighbor(func(id uint32, conn *Conn) bool {
				syncList = append(syncList, p.getNodeHash(id))
				return true
			})
			syncList = append(syncList, p.getNodeHash(currId))
			m.UnixNano = t.UnixNano()
			data, _ := json.Marshal(syncList)
			m.Data = data
			p.broadcast(0, m)
		}
	}()
}

func (p *RouterBFS) getNodeHash(id uint32) *SyncMsg {
	var sIds []uint32
	if id == p.node.NodeId() {
		sIds = make([]uint32, 0, p.node.LenConn())
		p.node.RangeConn(func(conn conn.Conn) bool {
			sIds = append(sIds, conn.RemoteId())
			return true
		})
	} else {
		sIds = p.nodeTable.GetSubNodes(id)
	}
	return &SyncMsg{
		Id:         id,
		SubNodeNum: uint32(len(sIds)),
		Hash:       sum(sIds),
	}
}

func sum(v []uint32) (n uint64) {
	for i := 0; i < len(v); i++ {
		n += uint64(v[i])
	}
	return
}
func equalSyncHash(a, b *SyncMsg) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Id == b.Id && a.SubNodeNum == b.SubNodeNum && a.Hash == b.Hash
}

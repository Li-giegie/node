package routerbfs

import (
	"context"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/errors"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/reply"
	"github.com/Li-giegie/node/pkg/router"
	"sync"
	"sync/atomic"
	"time"
)

func NewRouterBFS(protoType uint8, node node.Server) *RouterBFS {
	p := &RouterBFS{
		node:          node,
		protoType:     protoType,
		nodeTable:     newNodeTable(),
		neighborTable: newNeighborTable(),
		init:          true,
	}
	node.GetRouter().ReroutingHandleFunc(p.CalcRoute)
	return p
}

type RouterBFS struct {
	protoType       uint8       //协议类型
	init            bool        // 是否第一次OnConnect
	node            node.Server // 当前节点
	idCounter       int64
	neighborACKWait sync.WaitGroup
	*nodeTable      //全部节点
	*neighborTable  //直连的协议节点

}

func (p *RouterBFS) OnConnect(c *conn.Conn) bool {
	if p.init {
		p.init = false
		p.node.RangeConn(func(c *conn.Conn) bool {
			go p.onConnect(c)
			return true
		})
	}
	go p.onConnect(c)
	return true
}

func (p *RouterBFS) onConnect(c *conn.Conn) {
	subId := c.RemoteId()
	p.nodeTable.AddNode(p.node.NodeId(), subId, c.ConnType())
	p.node.GetRouter().RemoveRoute(c.RemoteId())
	p.broadcast(0, &ProtoMsg{
		Id:     atomic.AddInt64(&p.idCounter, 1),
		Action: Action_Update,
		SrcId:  p.node.NodeId(),
		Paths:  []uint32{p.node.NodeId(), subId},
		Data: (&UpdateMsg{
			{
				Action:  UpdateAction_AddNode,
				RootId:  p.node.NodeId(),
				SubId:   subId,
				SubType: c.ConnType(),
			},
		}).Encode(),
	})
	if c.ConnType() == conn.TypeServer {
		c.SendType(p.protoType, (&ProtoMsg{
			Id:     atomic.AddInt64(&p.idCounter, 1),
			Action: Action_NeighborASK,
			SrcId:  p.node.NodeId(),
			Paths:  []uint32{p.node.NodeId()},
		}).Encode())
	}
}

func (p *RouterBFS) OnMessage(r *reply.Reply, msg *message.Message) bool {
	proto, err := p.validMessage(msg)
	if err != nil {
		return true
	}
	switch proto.Action {
	case Action_NeighborASK: //邻居捂手请求 走单播
		r.GetConn().SendType(p.protoType, (&ProtoMsg{
			Id:     atomic.AddInt64(&p.idCounter, 1),
			Action: Action_NeighborACK,
			SrcId:  p.node.NodeId(),
			Paths:  []uint32{p.node.NodeId()},
		}).Encode())
	case Action_NeighborACK: //邻居握手响应 走单播，并携带已知信息，去拉取邻居节点的信息
		p.neighborTable.AddNeighbor(proto.SrcId, r.GetConn())
		r.GetConn().SendType(p.protoType, (&ProtoMsg{
			Id:     atomic.AddInt64(&p.idCounter, 1),
			Action: Action_PullNode,
			SrcId:  p.node.NodeId(),
			Paths:  []uint32{p.node.NodeId()},
			Data:   p.nodeTable.RootList(proto.SrcId).Encode(),
		}).Encode())
	case Action_PullNode: // 邻居响应当前节点的已知节点
		var hasNodeList List
		if err = hasNodeList.Decode(proto.Data); err != nil {
			return false
		}
		m := hasNodeList.Map()
		var push PushMsg
		if pl := p.nodeTable.Len() - len(hasNodeList); pl > 0 {
			push = make(PushMsg, 0, pl)
		}
		paths := make([]uint32, 1, p.nodeTable.Len())
		paths[0] = p.node.NodeId()
		p.nodeTable.Range(func(rootId uint32, item *NodeTableEmpty) bool {
			if rootId != proto.SrcId && rootId != p.node.NodeId() {
				paths = append(paths, rootId)
			}
			if _, ok := m[rootId]; !ok {
				var entry PushMsgEntry
				entry.Root = rootId
				entry.SubEntry = make([]PushMsgSubEntry, 0, len(item.Cache))
				for sId, sType := range item.Cache {
					entry.SubEntry = append(entry.SubEntry, PushMsgSubEntry{
						SubId:   sId,
						SubType: sType,
					})
				}
				push = append(push, entry)
			}
			return true
		})
		if len(push) > 0 {
			r.GetConn().SendType(p.protoType, (&ProtoMsg{
				Id:     atomic.AddInt64(&p.idCounter, 1),
				Action: Action_PushNode,
				SrcId:  p.node.NodeId(),
				Paths:  paths,
				Data:   push.Encode(),
			}).Encode())
		}
	case Action_PushNode:
		var push PushMsg
		if err = push.Decode(proto.Data); err != nil {
			return false
		}
		for _, entry := range push {
			for _, subEntry := range entry.SubEntry {
				p.nodeTable.AddNode(entry.Root, subEntry.SubId, subEntry.SubType)
			}
		}
		p.nodeTable.Range(func(rootId uint32, item *NodeTableEmpty) bool {
			p.addRoute(rootId)
			for sId := range item.Cache {
				p.addRoute(sId)
			}
			return true
		})
		p.broadcast(msg.Hop, proto)
	case Action_Update:
		var updateList UpdateMsg
		if err = updateList.Decode(proto.Data); err != nil {
			return false
		}
		var success = make(UpdateMsg, 0, len(updateList))
		for _, update := range updateList {
			if update.RootId == p.node.NodeId() {
				continue
			}
			switch update.Action {
			case UpdateAction_AddNode:
				if p.AddNode(update.RootId, update.SubId, update.SubType) {
					success = append(success, update)
				}
			case UpdateAction_RemoveNode:
				if p.RemoveNode(update.RootId, update.SubId, update.SubType) {
					success = append(success, update)
				}
			}
		}
		if len(success) > 0 {
			for _, updateMsg := range success {
				if updateMsg.Action == UpdateAction_AddNode {
					p.addRoute(updateMsg.SubId)
				} else {
					p.removeRoute(updateMsg.SubId)
				}
			}
			proto.Data = success.Encode()
			p.broadcast(msg.Hop, proto)
		}
	case Action_SyncHash:
		var syncHash SyncMsg
		if err = syncHash.Decode(proto.Data); err != nil {
			return false
		}
		localHash := p.getNodeHash(syncHash.Id)
		if *localHash != syncHash {
			if syncHash.SubNodeNum == 0 {
				p.RemoveRootNode(syncHash.Id)
				p.nodeTable.Range(func(rootId uint32, item *NodeTableEmpty) bool {
					p.addRoute(rootId)
					for sId := range item.Cache {
						p.addRoute(sId)
					}
					return true
				})
				return false
			}
			r.GetConn().SendType(p.protoType, (&ProtoMsg{
				Id:     atomic.AddInt64(&p.idCounter, 1),
				Action: Action_SyncQueryNode,
				SrcId:  p.node.NodeId(),
				Paths:  []uint32{p.node.NodeId()},
				Data:   proto.Data,
			}).Encode())
		}
		p.broadcast(msg.Hop, proto)
	case Action_SyncQueryNode:
		var syncHash SyncMsg
		if err = syncHash.Decode(proto.Data); err != nil {
			return false
		}
		dstHash := p.getNodeHash(syncHash.Id)
		if *dstHash == syncHash {
			var push PushMsgEntry
			p.nodeTable.RangeSubNode(syncHash.Id, func(m map[uint32]conn.Type) {
				push.Root = syncHash.Id
				push.SubEntry = make([]PushMsgSubEntry, 0, len(m))
				for u, nodeType := range m {
					push.SubEntry = append(push.SubEntry, PushMsgSubEntry{
						SubId:   u,
						SubType: nodeType,
					})
				}
			})
			if len(push.SubEntry) > 0 {
				r.GetConn().SendType(p.protoType, (&ProtoMsg{
					Id:     atomic.AddInt64(&p.idCounter, 1),
					Action: Action_SyncNode,
					SrcId:  p.node.NodeId(),
					Paths:  *p.nodeTable.RootList(proto.SrcId),
					Data:   push.Encode(),
				}).Encode())
			}
			return false
		}
		p.broadcast(msg.Hop, proto)
	case Action_SyncNode:
		var pushEntry PushMsgEntry
		if err = pushEntry.Decode(proto.Data); err != nil {
			return false
		}
		p.nodeTable.RemoveRootNode(pushEntry.Root)
		for _, entry := range pushEntry.SubEntry {
			p.nodeTable.AddNode(pushEntry.Root, entry.SubId, entry.SubType)
		}
		p.nodeTable.Range(func(rootId uint32, item *NodeTableEmpty) bool {
			p.addRoute(rootId)
			for sId := range item.Cache {
				p.addRoute(sId)
			}
			return true
		})
	}
	return false
}

func (p *RouterBFS) OnClose(c *conn.Conn, err error) bool {
	subId := c.RemoteId()
	p.nodeTable.RemoveNode(p.node.NodeId(), subId, c.ConnType())
	p.removeRoute(subId)
	if c.ConnType() == conn.TypeServer {
		p.neighborTable.DeleteNeighbor(subId)
	}
	p.broadcast(0, &ProtoMsg{
		Action: Action_Update,
		Id:     atomic.AddInt64(&p.idCounter, 1),
		SrcId:  p.node.NodeId(),
		Paths:  []uint32{p.node.NodeId(), subId},
		Data: (&UpdateMsg{
			{
				Action:  UpdateAction_RemoveNode,
				RootId:  p.node.NodeId(),
				SubId:   subId,
				SubType: c.ConnType(),
			},
		}).Encode(),
	})
	return true
}

func (p *RouterBFS) validMessage(msg *message.Message) (*ProtoMsg, error) {
	if msg.Hop >= 254 || msg.Hop >= p.node.RouteHop() && p.node.RouteHop() > 0 {
		return nil, errors.New("hop overflow Loop messages")
	}
	proto := new(ProtoMsg)
	err := proto.Decode(msg.Data)
	if err != nil {
		return nil, err
	}
	if err = proto.Valid(); err != nil {
		return proto, err
	}
	for _, path := range proto.Paths {
		if path == p.node.NodeId() {
			return proto, errors.New("invalid path")
		}
	}
	proto.Paths = append(proto.Paths, p.node.NodeId())
	return proto, nil
}

func (p *RouterBFS) addRoute(id uint32) (*router.RouteEmpty, bool) {
	if id == p.node.NodeId() {
		return nil, false
	}
	if _, ok := p.node.GetConn(id); ok {
		return nil, true
	}
	empty, ok := p.CalcRoute(id)
	if !ok {
		return nil, false
	}
	p.node.GetRouter().AddRoute(empty.Dst, empty.Via, empty.Hop, empty.UnixNano, empty.Paths)
	return empty, true
}

func (p *RouterBFS) copyPath(paths []uint32, v uint32) []uint32 {
	newPath := make([]uint32, len(paths)+1)
	copy(newPath, paths)
	newPath[len(paths)] = v
	return newPath
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

func (p *RouterBFS) broadcast(hop uint8, msg *ProtoMsg) {
	num := p.neighborTable.Len()
	if num == 0 {
		return
	}
	var filterIds = make(map[uint32]struct{}, len(msg.Paths))
	for _, path := range msg.Paths {
		filterIds[path] = struct{}{}
	}
	var conns = make([]*conn.Conn, 0, num)
	var connIds = make([]uint32, 0, num)
	p.RangeNeighbor(func(id uint32, conn *conn.Conn) bool {
		if _, ok := filterIds[id]; !ok {
			connIds = append(connIds, id)
			conns = append(conns, conn)
		}
		return true
	})
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
		p.RangeSubNode(current.node, func(empty map[uint32]conn.Type) {
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
	paths := p.BFSSearch(p.node.NodeId(), dst, p.node.RouteHop())
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
		ticker := time.NewTicker(timeout)
		defer ticker.Stop()
		go func() {
			<-ctx.Done()
			ticker.Stop()
		}()
		for range ticker.C {
			msg := &ProtoMsg{
				Id:     atomic.AddInt64(&p.idCounter, 1),
				Action: Action_SyncHash,
				SrcId:  p.node.NodeId(),
				Paths:  []uint32{p.node.NodeId()},
				Data:   p.getNodeHash(p.node.NodeId()).Encode(),
			}
			p.broadcast(p.protoType, msg)
		}
	}()
}

func (p *RouterBFS) getNodeHash(id uint32) *SyncMsg {
	var sIds []uint32
	if id == p.node.NodeId() {
		sIds = make([]uint32, 0, p.node.LenConn())
		p.node.RangeConn(func(conn *conn.Conn) bool {
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

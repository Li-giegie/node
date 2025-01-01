package routerbfs

import (
	"context"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/handler"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"github.com/Li-giegie/node/pkg/router"
	"sync/atomic"
	"time"
)

type Node interface {
	NodeId() uint32
	GetAllConn() []conn.Conn
	GetConn(id uint32) (conn.Conn, bool)
	GetRouter() router.Router
}

func NewRouterBFS(protoType uint8, node Node, MaxRouteHop uint8, requestTimeout, clearExistCacheTime time.Duration) *RouterBFS {
	p := &RouterBFS{
		node:           node,
		protoType:      protoType,
		maxRouteHop:    MaxRouteHop,
		fullNode:       newFullNodeContainer(),
		existCache:     newExistContainer(clearExistCacheTime),
		protoNode:      newProtoNodeContainer(),
		requestTimeout: requestTimeout,
	}
	node.GetRouter().ReroutingHandleFunc(p.ReroutingHandleFunc)
	return p
}

type RouterBFS struct {
	handler.Empty
	requestTimeout time.Duration
	protoType      uint8
	idCounter      uint32
	maxRouteHop    uint8
	node           Node
	fullNode       *fullNodeContainer
	protoNode      *protoNodeContainer
	existCache     *existContainer
	once           bool
	unixNano       int64
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
	p.unixNano = time.Now().UnixNano()
	p.once = true
	conns := p.node.GetAllConn()
	for i := 0; i < len(conns); i++ {
		go p.onConnect(conns[i])
	}
}

func (p *RouterBFS) onConnect(conn conn.Conn) {
	unixNao := time.Now().UnixNano()
	subId := conn.RemoteId()
	p.fullNode.Add(p.node.NodeId(), subId, unixNao)
	p.node.GetRouter().RemoveRoute(subId, unixNao)
	p.broadcast(p.newPMsg(Action_AddNode, []*NodeInfo{
		{
			RootNodeId: p.node.NodeId(),
			SubNodeInfo: []*SubInfo{
				{
					Id:      subId,
					UnixNao: unixNao,
				},
			},
		},
	}), 0, subId)
	// 如果是桥接类型节点则发送本地所有节点和本地路由
	_ctx, cancel := context.WithTimeout(context.Background(), p.requestTimeout)
	defer cancel()
	res, code, err := conn.RequestType(_ctx, p.protoType, p.newPMsg(Action_Query, nil).Encode())
	if err != nil || code != 200 {
		return
	}
	var proto ProtoMsg
	if err = proto.Decode(res); err != nil {
		return
	}
	p.protoNode.Add(subId, conn)
	p.existCache.Remove(subId)
	p.addNodes(proto.NInfo)
	proto.Action = Action_AddNode
	p.broadcast(&proto, 0, subId)
}

func (p *RouterBFS) OnMessage(r responsewriter.ResponseWriter, msg *message.Message) {
	if msg.Hop >= 254 || msg.Hop >= p.maxRouteHop && p.maxRouteHop > 0 {
		return
	}
	var proto ProtoMsg
	// 从ctx.Data中解码协议消息
	if err := proto.Decode(msg.Data); err != nil {
		return
	}
	// 每条消息只会被处理一次，消息Id如果处理过存在，不再处理
	if p.existCache.ExistOrStore(proto.SrcId, proto.Id, time.Now().UnixNano()) {
		return
	}
	switch proto.Action {
	case Action_Query:
		proto.Action = Action_Reply
		proto.SrcId = p.node.NodeId()
		proto.NInfo = p.fullNode.GetAllNodeInfo()
		proto.SrcId = msg.DestId
		_ = r.Response(200, proto.Encode())
		return
	case Action_AddNode:
		p.addNodes(proto.NInfo)
	case Action_RemoveNode:
		p.removeNodes(proto.NInfo)
	default:
		return
	}
	p.broadcast(&proto, msg.Hop, msg.SrcId, proto.SrcId)
}

func (p *RouterBFS) OnClose(conn conn.Conn, err error) {
	id := conn.RemoteId()
	nodes := []*NodeInfo{{RootNodeId: p.node.NodeId(), SubNodeInfo: []*SubInfo{{id, time.Now().UnixNano()}}}}
	p.protoNode.Remove(id)
	p.removeNodes(nodes)
	p.broadcast(p.newPMsg(Action_RemoveNode, nodes), 0)
}

func (p *RouterBFS) addNodes(infos []*NodeInfo) {
	for _, info := range infos {
		for _, subInfo := range info.SubNodeInfo {
			p.fullNode.Add(info.RootNodeId, subInfo.Id, subInfo.UnixNao)
		}
	}
	p.addRoute(infos)
}

func (p *RouterBFS) removeNodes(infos []*NodeInfo) {
	for _, info := range infos {
		for _, subInfo := range info.SubNodeInfo {
			p.existCache.Remove(subInfo.Id)
			p.fullNode.Remove(info.RootNodeId, subInfo.Id, subInfo.UnixNao)
		}
	}
	p.removeRoute(infos)
}

func (p *RouterBFS) addRoute(infos []*NodeInfo) {
	route := p.node.GetRouter()
	curId := p.node.NodeId()
	for _, info := range infos {
		if info.RootNodeId == curId {
			continue
		}
		if _, ok := p.node.GetConn(info.RootNodeId); ok {
			unixNano, ok := p.fullNode.GetNodeUnixNano(curId, info.RootNodeId)
			if !ok {
				continue
			}
			for _, subInfo := range info.SubNodeInfo {
				if subInfo.Id == curId {
					continue
				}
				if _, ok = p.node.GetConn(subInfo.Id); ok {
					continue
				}
				route.AddRoute(subInfo.Id, info.RootNodeId, 2, subInfo.UnixNao, []*router.RoutePath{
					{Id: curId, UnixNano: p.unixNano},
					{Id: info.RootNodeId, UnixNano: unixNano},
					{Id: subInfo.Id, UnixNano: subInfo.UnixNao},
				})
			}
			continue
		}
		paths := p.BFSSearch(curId, info.RootNodeId, p.maxRouteHop)
		lastIndex := len(paths)
		if lastIndex < 2 {
			continue
		}
		route.AddRoute(info.RootNodeId, paths[1].Id, uint8(lastIndex-1), paths[lastIndex-1].UnixNano, paths)
		for _, subInfo := range info.SubNodeInfo {
			if subInfo.Id == curId {
				continue
			}
			if _, ok := p.node.GetConn(subInfo.Id); ok {
				continue
			}
			subPaths := make([]*router.RoutePath, lastIndex+1)
			copyRoutePath(subPaths, paths)
			subPaths[lastIndex] = &router.RoutePath{Id: subInfo.Id, UnixNano: subInfo.UnixNao}
			route.AddRoute(subInfo.Id, paths[1].Id, uint8(lastIndex), subInfo.UnixNao, subPaths)
		}
	}
}

func (p *RouterBFS) removeRoute(infos []*NodeInfo) {
	route := p.node.GetRouter()
	curId := p.node.NodeId()
	now := time.Now().UnixNano()
	for _, info := range infos {
		if info.RootNodeId != curId {
			if _, exist := p.node.GetConn(info.RootNodeId); exist {
				route.RemoveRoute(info.RootNodeId, now)
			} else {
				paths := p.BFSSearch(curId, info.RootNodeId, p.maxRouteHop)
				length := len(paths)
				if length > 2 {
					route.AddRoute(info.RootNodeId, paths[1].Id, uint8(length-1), paths[length-1].UnixNano, paths)
				} else {
					route.RemoveRoute(info.RootNodeId, now)
					route.RemoveRouteWithVia(info.RootNodeId, now)
					route.RemoveRouteWithPath(info.RootNodeId, now)
				}
			}
		}
		for _, subInfo := range info.SubNodeInfo {
			if subInfo.Id == curId {
				continue
			}
			if _, exist := p.node.GetConn(subInfo.Id); exist {
				route.RemoveRoute(subInfo.Id, now)
				continue
			}
			paths := p.BFSSearch(curId, subInfo.Id, p.maxRouteHop)
			length := len(paths)
			if length > 2 {
				route.AddRoute(subInfo.Id, paths[1].Id, uint8(length-1), paths[length-1].UnixNano, paths)
			} else {
				route.RemoveRoute(subInfo.Id, subInfo.UnixNao)
				route.RemoveRouteWithVia(subInfo.Id, subInfo.UnixNao)
				route.RemoveRouteWithPath(subInfo.Id, subInfo.UnixNao)
			}
		}
	}
}

func copyRoutePath(dst []*router.RoutePath, src []*router.RoutePath) {
	for i, path := range src {
		dst[i] = &router.RoutePath{
			Id:       path.Id,
			UnixNano: path.UnixNano,
		}
	}
}

func (p *RouterBFS) newPMsg(action uint8, info []*NodeInfo) *ProtoMsg {
	m := ProtoMsg{
		Id:     atomic.AddUint32(&p.idCounter, 1),
		SrcId:  p.node.NodeId(),
		Action: action,
		NInfo:  info,
	}
	p.existCache.ExistOrStore(p.node.NodeId(), m.Id, time.Now().UnixNano())
	return &m
}

func (p *RouterBFS) broadcast(m *ProtoMsg, hop uint8, filterIds ...uint32) {
	data := m.Encode()
	p.protoNode.Range(func(id uint32, conn conn.Conn) {
		for _, filterId := range filterIds {
			if filterId == id {
				return
			}
		}
		_ = conn.SendMessage(&message.Message{
			Type:   p.protoType,
			Hop:    hop,
			SrcId:  conn.LocalId(),
			DestId: conn.RemoteId(),
			Data:   data,
		})
	})
}

type bfsResult struct {
	node  *router.RoutePath
	paths []*router.RoutePath
}

// BFSSearch 搜索起点到终端的路径，src起点，dst终点，maxDeep最大深度，返回起点到终点的全部路径，bool是否存在
func (p *RouterBFS) BFSSearch(src, dst uint32, maxDeep uint8) []*router.RoutePath {
	if src == dst {
		return nil
	}
	queue := []*bfsResult{{node: &router.RoutePath{Id: src}, paths: []*router.RoutePath{{Id: src, UnixNano: 0}}}}
	existTab := map[uint32]struct{}{src: {}}
	var unixNano int64
	var ok bool
	var subId uint32
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if len(current.paths) >= int(maxDeep) {
			return nil
		}
		result := p.fullNode.Find(current.node.Id, func(subTab map[uint32]int64) interface{} {
			if unixNano, ok = subTab[dst]; ok {
				return append(current.paths, &router.RoutePath{Id: dst, UnixNano: unixNano})
			}
			for subId, unixNano = range subTab {
				if _, ok = existTab[subId]; ok {
					continue
				}
				existTab[subId] = struct{}{}
				queue = append(queue, &bfsResult{
					node:  &router.RoutePath{Id: subId, UnixNano: unixNano},
					paths: append(current.paths, &router.RoutePath{Id: subId, UnixNano: unixNano})},
				)
			}
			return nil
		})
		if result != nil {
			return result.([]*router.RoutePath)
		}
	}
	return nil
}

func (p *RouterBFS) ReroutingHandleFunc(dst uint32) (*router.RouteEmpty, bool) {
	paths := p.BFSSearch(p.node.NodeId(), dst, p.maxRouteHop)
	if len(paths) < 2 {
		return nil, false
	}
	empty := &router.RouteEmpty{
		Dst:      dst,
		Via:      paths[1].Id,
		Hop:      uint8(len(paths) - 1),
		UnixNano: paths[len(paths)-1].UnixNano,
		Paths:    paths,
	}
	p.node.GetRouter().AddRoute(empty.Dst, empty.Via, empty.Hop, empty.UnixNano, empty.Paths)
	return empty, true
}

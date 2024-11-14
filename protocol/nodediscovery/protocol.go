// package nodediscovery 动态节点路由发现

package nodediscovery

import (
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	"sync"
	"sync/atomic"
	"time"
)

type Node interface {
	Id() uint32
	GetAllConn() []iface.Conn
	GetAllId() []uint32
	GetConn(id uint32) (iface.Conn, bool)
	iface.Router
}

type NodeDiscovery struct {
	ProtoMsgType uint8
	Node         Node
	MaxHop       uint8
	counter      uint32
	existTab     *existTab
}

func NewNodeDiscovery(ProtoMsgType uint8, node Node, maxHop, existTabMaxRecordNum uint8) *NodeDiscovery {
	if existTabMaxRecordNum == 0 {
		existTabMaxRecordNum = 10
	}
	return &NodeDiscovery{
		ProtoMsgType: ProtoMsgType,
		Node:         node,
		MaxHop:       maxHop,
		existTab: &existTab{
			m:            make(map[uint32]*existContainer),
			maxRecordNum: existTabMaxRecordNum,
		},
	}
}

func (q *NodeDiscovery) createProtoMsg(action uint8, nodeList []uint32, routes []*Route) *ProtoMsg {
	return &ProtoMsg{
		Id:       atomic.AddUint32(&q.counter, 1),
		SrcId:    q.Node.Id(),
		Action:   action,
		NodeList: nodeList,
		Routes:   routes,
		Counter:  0,
		UnixNano: time.Now().UnixNano(),
	}
}

func (q *NodeDiscovery) broadcast(m *ProtoMsg, filter ...uint32) {
	data := m.Encode()
	conns := q.Node.GetAllConn()
	q.existTab.Add(m.SrcId, m.Id)
	var ok bool
	for _, conn := range conns {
		if conn.NodeType() != uint8(node.NodeType_Bridge) {
			continue
		}
		ok = true
		for _, u := range filter {
			if u == conn.RemoteId() {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		q.send(conn, data)
	}
}

func (q *NodeDiscovery) send(conn iface.Conn, data []byte) {
	_, _ = conn.WriteMsg(&message.Message{
		Type:   q.ProtoMsgType,
		SrcId:  conn.LocalId(),
		DestId: conn.RemoteId(),
		Data:   data,
	})
}

func (q *NodeDiscovery) OnConnection(conn iface.Conn) {
	q.Node.RemoveRouteWithDst(conn.RemoteId())
	q.broadcast(q.createProtoMsg(ACTION_PUSH, []uint32{conn.RemoteId()}, nil), conn.RemoteId())
	if conn.NodeType() == uint8(node.NodeType_Bridge) {
		q.send(conn, q.createProtoMsg(ACTION_PULL, nil, nil).Encode())
	}
}

func (q *NodeDiscovery) OnCustomMessage(ctx iface.Context) {
	if ctx.Type() != q.ProtoMsgType {
		return
	}
	ctx.Stop()
	var m ProtoMsg
	var exist bool
	if err := m.Decode(ctx.Data()); err != nil {
		return
	}
	if m.Counter >= 255 || q.MaxHop > 0 && m.Counter > q.MaxHop {
		return
	}
	if exist = q.existTab.Exist(m.SrcId, m.Id); exist {
		return
	}
	q.existTab.Add(m.SrcId, m.Id)
	m.Counter++
	switch m.Action {
	case ACTION_PULL:
		tmpIds := q.Node.GetAllId()
		routes := q.getRoutes()
		if len(tmpIds) == 0 && len(routes) == 0 {
			return
		}
		ids := make([]uint32, 0, len(tmpIds))
		for i := 0; i < len(tmpIds); i++ {
			if tmpIds[i] != ctx.SrcId() {
				ids = append(ids, tmpIds[i])
			}
		}
		if len(ids) > 0 {
			_ = ctx.CustomReply(q.ProtoMsgType, q.createProtoMsg(ACTION_PUSH, ids, routes).Encode())
		}
	case ACTION_PUSH:
		rs := make([]*Route, 0, len(m.Routes))
		for _, route := range m.Routes {
			route.Hop++
			if q.addRoute(route.Dst, ctx.SrcId(), route.Hop, time.Duration(m.UnixNano)) {
				rs = append(rs, &Route{
					Dst: route.Dst,
					Hop: route.Hop,
				})
			}
		}
		ids := make([]uint32, 0, len(m.NodeList))
		for _, dst := range m.NodeList {
			if q.addRoute(dst, ctx.SrcId(), m.Counter, time.Duration(m.UnixNano)) {
				ids = append(ids, dst)
			}
		}
		m.NodeList = ids
		m.Routes = rs
		filterNodes := make([]uint32, 0, len(m.NodeList)+2)
		filterNodes = append(filterNodes, m.NodeList...)
		filterNodes = append(filterNodes, ctx.SrcId())
		filterNodes = append(filterNodes, m.SrcId)
		q.broadcast(&m, filterNodes...)
	case ACTION_DELETE:
		q.broadcast(&m, ctx.SrcId(), m.SrcId)
		// 删除路由前，保存经过该路由的所有节点信息，如果当前节点可以到的则广播路由
		var viaDelRoute []uint32
		for _, dst := range m.NodeList {
			tmpViaRoute := q.Node.GetRouteWithVia(dst)
			if q.Node.RemoveRouteWithCallback(dst, func(info iface.RouteInfo) (isDel bool) {
				return m.UnixNano > int64(info.Activation())
			}) {
				viaDelRoute = append(viaDelRoute, tmpViaRoute...)
				q.Node.RemoveRouteWithVia(dst)
				q.existTab.Delete(dst)
			}
		}
		otherRoute := make([]uint32, 0)
		for _, u := range viaDelRoute {
			if _, exist = q.Node.GetConn(u); exist {
				otherRoute = append(otherRoute, u)
			}
		}
		if len(otherRoute) > 0 {
			q.broadcast(q.createProtoMsg(ACTION_PUSH, otherRoute, nil))
		}
	case ACTION_QUERY:
		var ids, noIds []uint32
		var rs []*Route
		for _, u := range m.NodeList {
			if _, exist = q.Node.GetConn(u); exist {
				ids = append(ids, u)
			} else {
				noIds = append(noIds, u)
			}
		}
		if len(ids) > 0 || len(rs) > 0 {
			_ = ctx.CustomReply(q.ProtoMsgType, q.createProtoMsg(ACTION_PUSH, ids, rs).Encode())
		}
		if len(noIds) > 0 {
			m.NodeList = noIds
			q.broadcast(&m, ctx.SrcId(), m.SrcId)
		}
	}
}

func (q *NodeDiscovery) addRoute(dst, via uint32, hop uint8, d time.Duration) bool {
	if q.Node.Id() == dst {
		return false
	} else if _, exist := q.Node.GetConn(dst); exist {
		return false
	}
	return q.Node.AddRoute(dst, via, hop, d)
}

func (q *NodeDiscovery) OnClose(conn iface.Conn, err error) {
	id := conn.RemoteId()
	viaIds := q.Node.GetRouteWithVia(id)
	q.Node.RemoveRouteWithDst(id)
	q.Node.RemoveRouteWithVia(id)
	q.broadcast(q.createProtoMsg(ACTION_DELETE, append(viaIds, id), nil))
	q.existTab.Delete(conn.RemoteId())
	if len(viaIds) > 0 {
		q.broadcast(q.createProtoMsg(ACTION_QUERY, viaIds, nil))
	}
}

func (q *NodeDiscovery) getRoutes() []*Route {
	var rs []*Route
	q.Node.RangeRoute(func(info iface.RouteInfo) {
		rs = append(rs, &Route{
			Dst: info.Dst(),
			Hop: info.Hop(),
		})
	})
	return rs
}

type existTab struct {
	m            map[uint32]*existContainer
	l            sync.RWMutex
	maxRecordNum uint8
}

func (e *existTab) Add(nodeId, ProtoMsgId uint32) {
	e.l.Lock()
	ec, ok := e.m[nodeId]
	if !ok {
		ec = &existContainer{Size: e.maxRecordNum, Record: make([]uint32, e.maxRecordNum)}
	}
	ec.Add(ProtoMsgId)
	e.m[nodeId] = ec
	e.l.Unlock()
}

func (e *existTab) Delete(nodeId uint32) {
	e.l.Lock()
	delete(e.m, nodeId)
	e.l.Unlock()
}

func (e *existTab) Exist(nodeId, ProtoMsgId uint32) bool {
	e.l.RLock()
	ec, ok := e.m[nodeId]
	e.l.RUnlock()
	if !ok {
		return false
	}
	return ec.Exist(ProtoMsgId)
}

type existContainer struct {
	Size   uint8
	Record []uint32
	Index  uint8
}

func (c *existContainer) Add(n uint32) {
	if c.Index < c.Size {
		c.Record[c.Index] = n
		c.Index++
		return
	}
	copy(c.Record, c.Record[1:])
	c.Record[c.Size-1] = n
}

func (c *existContainer) Remove(n uint32) {
	for i := uint8(0); i < c.Index; i++ {
		if n == c.Record[i] {
			copy(c.Record, c.Record[:i])
			copy(c.Record, c.Record[i+1:])
			c.Index--
			break
		}
	}
}

func (c *existContainer) Exist(n uint32) bool {
	for i := uint8(0); i < c.Index; i++ {
		if c.Record[i] == n {
			return true
		}
	}
	return false
}

func (c *existContainer) GetAll() []uint32 {
	return c.Record[:c.Index]
}

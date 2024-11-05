package nodediscovery

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	"sync"
	"sync/atomic"
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
	existTab     sync.Map
}

func (q *NodeDiscovery) CreateID() string {
	return fmt.Sprintf("%d%d", q.Node.Id(), atomic.AddUint32(&q.counter, 1))
}

func (q *NodeDiscovery) Broadcast(m *ProtoMsg, filter ...uint32) {
	data := m.Encode()
	conns := q.Node.GetAllConn()
	q.existTab.Store(m.PId, struct{}{})
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
		_, _ = conn.WriteMsg(&message.Message{
			Type:   q.ProtoMsgType,
			SrcId:  conn.LocalId(),
			DestId: conn.RemoteId(),
			Data:   data,
		})
	}
}

func (q *NodeDiscovery) OnConnection(conn iface.Conn) {
	q.Node.RemoveRouteWithDst(conn.RemoteId())
	if conn.NodeType() == uint8(node.NodeType_Bridge) {
		q.Broadcast(&ProtoMsg{
			PId:    q.CreateID(),
			Action: ACTION_GET,
		})
		ids := q.Node.GetAllId()
		rs := q.GetRoutes()
		if len(ids) > 0 || len(rs) > 0 {
			q.Broadcast(&ProtoMsg{
				PId:      q.CreateID(),
				Action:   ACTION_PUSH,
				NodeList: ids,
				Routes:   rs,
			})
		}
	} else {
		q.Broadcast(&ProtoMsg{
			PId:      q.CreateID(),
			Action:   ACTION_PUSH,
			NodeList: []uint32{conn.RemoteId()},
		}, conn.RemoteId())
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
	if _, exist = q.existTab.Load(m.PId); exist {
		return
	}
	if q.MaxHop > 0 && m.Counter > q.MaxHop || m.Counter >= 255 {
		ctx.Stop()
		return
	}
	m.Counter++
	switch m.Action {
	case ACTION_GET:
		m.Action = ACTION_PUSH
		m.PId = q.CreateID()
		m.NodeList = q.Node.GetAllId()
		m.Routes = q.GetRoutes()
		m.Counter = 0
		_ = ctx.CustomReply(q.ProtoMsgType, m.Encode())
		return
	case ACTION_PUSH:
		for _, dst := range m.NodeList {
			if dst == q.Node.Id() {
				continue
			} else if _, exist = q.Node.GetConn(dst); exist {
				continue
			}
			q.Node.AddRoute(dst, ctx.SrcId(), m.Counter)
		}
		for _, route := range m.Routes {
			if route.Dst == q.Node.Id() {
				continue
			} else if _, exist = q.Node.GetConn(route.Dst); exist {
				continue
			}
			q.Node.AddRoute(route.Dst, ctx.SrcId(), route.Hop)
		}
	case ACTION_DELETE:
		// 删除路由前，保存经过该路由的所有节点信息，如果当前节点可以到的则广播路由
		viaDelRoute := make([]uint32, 0)
		for _, dst := range m.NodeList {
			viaDelRoute = append(viaDelRoute, q.Node.GetRouteWithVia(dst)...)
			q.Node.RemoveRouteWithDst(dst)
			q.Node.RemoveRouteWithVia(dst)
		}
		otherRoute := make([]uint32, 0)
		for _, u := range viaDelRoute {
			if _, exist = q.Node.GetConn(u); exist {
				otherRoute = append(otherRoute, u)
			}
		}
		if len(otherRoute) > 0 {
			q.Broadcast(&ProtoMsg{
				PId:      q.CreateID(),
				Action:   ACTION_PUSH,
				NodeList: otherRoute,
			})
		}
	default:
		return
	}
	q.Broadcast(&m, ctx.SrcId())
	return
}

func (q *NodeDiscovery) OnClose(conn iface.Conn, err error) {
	id := conn.RemoteId()
	viaDelRoute := q.Node.GetRouteWithVia(id)
	q.Node.RemoveRouteWithDst(id)
	q.Node.RemoveRouteWithVia(id)
	q.Broadcast(&ProtoMsg{
		PId:      q.CreateID(),
		Action:   ACTION_DELETE,
		NodeList: []uint32{id},
		Counter:  0,
	})
	// 删除路由前，保存经过该路由的所有节点信息，如果当前节点可以到的则广播路由
	otherRoute := make([]uint32, 0)
	connIds := q.Node.GetAllId()
	var exist bool
	for _, via := range viaDelRoute {
		exist = false
		for _, connId := range connIds {
			if via == connId {
				exist = true
				break
			}
		}
		if exist {
			otherRoute = append(otherRoute, via)
		}
	}
	if len(otherRoute) > 0 {
		q.Broadcast(&ProtoMsg{
			PId:      q.CreateID(),
			Action:   ACTION_PUSH,
			NodeList: otherRoute,
		})
	}
}

func (q *NodeDiscovery) GetRoutes() []*Route {
	var rs []*Route
	q.Node.RangeRoute(func(id uint64, dst uint32, via uint32, hop uint8) {
		rs = append(rs, &Route{
			Dst: dst,
			Hop: hop,
		})
	})
	return rs
}

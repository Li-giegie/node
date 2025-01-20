package routerbfsv2

import (
	"github.com/Li-giegie/node/pkg/conn"
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
type BFSRouter struct {
	protoType uint8
	PeerTab
	NodeTab
	Node
}

func (r *BFSRouter) OnConnect(conn conn.Conn) {

}

func (r *BFSRouter) OnMessage(writer responsewriter.ResponseWriter, msg *message.Message) {
	p, err := DecodeProtoMsg(msg.Data)
	if err != nil {
		return
	}
	for _, path := range p.Paths {
		if path == r.NodeId() {
			return
		}
	}
	p.Paths = append(p.Paths, r.NodeId())
	if c, ok := r.PeerTab.Get(p.SrcId); !ok || c.UpdateAt > p.UnixNano {
		return
	} else {
		c.UpdateAt = p.UnixNano
	}
	route := r.Node.GetRouter()
	switch p.Action {
	case Action_Open: //单播
		r.PeerTab.Add(p.SrcId, &Conn{Conn: writer.GetConn()})
		resp := ProtoMsg{
			Action:   Action_Refresh,
			SrcId:    r.NodeId(),
			Paths:    []uint32{r.NodeId()},
			UnixNano: time.Now().UnixNano(),
		}
		resp.Routes = make([]*router.RouteEmpty, 0, route.RouteLen())
		route.RangeRoute(func(empty *router.RouteEmpty) bool {
			resp.Routes = append(resp.Routes, empty)
			return true
		})
		resp.Nodes = make([]uint32, 0, r.LenConn())
		r.RangeConn(func(conn conn.Conn) bool {
			if conn.RemoteId() != p.SrcId {
				resp.Nodes = append(resp.Nodes, conn.RemoteId())
			}
			return true
		})
		if len(resp.Routes) == 0 && len(resp.Nodes) == 0 {
			return
		}
		if err = writer.GetConn().SendType(r.protoType, resp.Encode()); err != nil {
			r.PeerTab.Remove(p.SrcId)
		}
	case Action_Refresh:
		var forwardEmpty []*router.RouteEmpty
		for _, Id := range p.Nodes {
			if _, ok := r.GetConn(Id); ok {
				continue
			}
			empty := &router.RouteEmpty{
				Dst:      Id,
				Via:      p.SrcId,
				Hop:      2,
				UnixNano: p.UnixNano,
				Paths:    []uint32{r.NodeId(), p.SrcId, Id},
			}
			if route.AddRoute(empty.Dst, empty.Via, empty.Hop, empty.UnixNano, empty.Paths) {
				forwardEmpty = append(forwardEmpty, empty)
			}
		}
		for _, newEmpty := range p.Routes {
			if _, ok := r.GetConn(newEmpty.Dst); ok {
				continue
			}
			newEmpty.Hop++
			newEmpty.Via = p.SrcId
			newEmpty.Paths = append([]uint32{r.NodeId()}, newEmpty.Paths...)
			if route.AddRoute(newEmpty.Dst, newEmpty.Via, newEmpty.Hop, newEmpty.UnixNano, newEmpty.Paths) {
				forwardEmpty = append(forwardEmpty, newEmpty)
			}
		}
		if len(forwardEmpty) > 0 {
			p.SrcId = r.NodeId()
			p.Routes = forwardEmpty
			p.Nodes = nil
			p.Action = Action_AddRoutes
			r.broadcastV2(p, msg.Hop)
		}
	case Action_AddRoutes:

	case Action_RemoveRoutes:

	default:

	}
}

func (p *BFSRouter) broadcastV2(msg *ProtoMsg, hop uint8) {
	p.PeerTab.Len()
	num := p.LenConn()
	if num == 0 {
		return
	}
	var filterIds = make(map[uint32]struct{}, len(msg.Paths))
	for _, path := range msg.Paths {
		filterIds[path] = struct{}{}
	}
	var conns = make([]*Conn, 0, num)
	var connIds = make([]uint32, 0, num)
	p.PeerTab.Range(func(id uint32, conn *Conn) bool {
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
		_ = conns[0].Conn.SendMessage(&message.Message{
			Type:   p.protoType,
			Hop:    hop,
			SrcId:  conns[0].Conn.LocalId(),
			DestId: conns[0].Conn.RemoteId(),
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
			SrcId:  c.Conn.LocalId(),
			DestId: c.Conn.RemoteId(),
			Data:   msg.Encode(),
		}
		_ = c.Conn.SendMessage(m)
		msg.Paths = msg.Paths[:l]
	}
}

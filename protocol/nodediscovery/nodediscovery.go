package nodediscovery

import (
	"context"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/net"
	"io"
	"log"
	"sync"
	"time"
)

type Conns interface {
	GetAll() []node.Conn
	Get(id uint32) (node.Conn, bool)
}

type Router interface {
	AddRoute(dst, next, parentNode uint32, hop uint16)
	DeleteRoute(dst, next, parentNode uint32, hop uint16) bool
	DeleteNextRoute(next uint32) bool
}

func NewNodeDiscoveryProtocol(id uint32, conns Conns, router Router, ProtocolMsgType uint8, out io.Writer) *NodeDiscoveryProtocol {
	var l *log.Logger
	if out != nil {
		l = log.New(out, "[DiscoveryNodeProtocol] ", log.Ldate|log.Ltime|log.Lmsgprefix)
	}
	p := &NodeDiscoveryProtocol{
		id:                              id,
		Conns:                           conns,
		Router:                          router,
		ProtocolMsgType:                 ProtocolMsgType,
		QueryEnableProtocolMaxNum:       3,
		QueryEnableProtocolIntervalStep: time.Millisecond * 500,
		l:                               new(sync.RWMutex),
		cache:                           make(map[uint32]*UniteNode),
		Logger:                          l,
	}
	return p
}

type NodeDiscoveryProtocol struct {
	id uint32
	Conns
	Router
	ProtocolMsgType                 uint8
	QueryEnableProtocolMaxNum       int
	QueryEnableProtocolIntervalStep time.Duration
	l                               *sync.RWMutex
	cache                           map[uint32]*UniteNode
	*log.Logger
}

func (n *NodeDiscoveryProtocol) StartTimingQueryEnableProtoNode(ctx context.Context, timeout time.Duration) (err error) {
	ok := true
	stopChan := make(chan struct{})
	go func() {
		protoMsg := NewProtoMsgWithType(ProtoMsgTyp_QueryEnable)
		for ok {
			time.Sleep(timeout)
			if err = n.Broadcast(protoMsg, false, 0, nil); err != nil {
				if n.Logger != nil {
					n.Logger.Println("err: StartTimingQueryEnableProtoNode Broadcast err", err)
				}
				stopChan <- struct{}{}
			}
		}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-stopChan:
		ok = false
		return
	}
}

func (n *NodeDiscoveryProtocol) Connection(conn node.Conn) {
	go func() {
		dstId := conn.RemoteId()
		protoMsg := NewProtoMsgWithOneNode(ProtoMsgTyp_SetNodes, true, dstId)
		protoMsg.ParentNodeId = n.id
		err := n.Broadcast(protoMsg, true, dstId, nil)
		if err != nil {
			if n.Logger != nil {
				n.Logger.Println("err: Connection Broadcast err", err)
			}
			return
		}
		protoMsg.Type = ProtoMsgTyp_QueryEnable
		protoMsg.Nodes = nil
		msg, err := n.NewMsgWithConn(conn, protoMsg)
		if err != nil {
			if n.Logger != nil {
				n.Logger.Println("err: Connection Create protoMsg err", err)
			}
			return
		}
		for i := 1; i <= n.QueryEnableProtocolMaxNum; i++ {
			if _, err = conn.WriteMsg(msg); err != nil {
				if n.Logger != nil {
					n.Logger.Println("err: Connection QueryEnableProtocol err", err)
				}
				return
			}
			time.Sleep(n.QueryEnableProtocolIntervalStep * time.Duration(i))
			if _, ok := n.Find(dstId); ok {
				if n.Logger != nil {
					n.Logger.Println("Connection query enable protocol node id", conn.RemoteId())
				}
				return
			}
		}
	}()
}

func (n *NodeDiscoveryProtocol) CustomHandle(ctx node.CustomContext) (next bool) {
	if n.ProtocolMsgType != ctx.Type() {
		return true
	}
	next = false
	go func() {
		srcId := ctx.SrcId()
		protoMsg, err := new(ProtoMsg).Decode(ctx.Data())
		if err != nil {
			if n.Logger != nil {
				n.Logger.Println("err: CustomHandle Decode ProtoMsg err", err)
			}
			return
		}
		switch protoMsg.Type {
		case ProtoMsgTyp_QueryEnable:
			if !n.AddNode(srcId) {
				if n.Logger != nil {
					n.Logger.Println("ProtoMsgTyp_QueryEnable err: CustomHandle Add Proto Node exist srcId", srcId)
				}
				return
			}
			protoMsg.Type = ProtoMsgTyp_ResponseEnable
			if err = n.Reply(ctx, protoMsg); err != nil {
				if n.Logger != nil {
					n.Logger.Println("ProtoMsgTyp_QueryEnable err: CustomHandle enable node success reply err srcId", srcId, err)
				}
			}
		case ProtoMsgTyp_ResponseEnable:
			if !n.AddNode(srcId) {
				if n.Logger != nil {
					n.Logger.Println("ProtoMsgTyp_ResponseEnable err: CustomHandle Add Proto Node exist srcId", srcId)
				}
				return
			}
			protoMsg.Type = ProtoMsgTyp_QueryNodes
			if err = n.Reply(ctx, protoMsg); err != nil {
				if n.Logger != nil {
					n.Logger.Println("ProtoMsgTyp_ResponseEnable err: CustomHandle enable node success reply err srcId", srcId, err)
				}
			}
		case ProtoMsgTyp_QueryNodes:
			protoMsg.IsAdd = true
			protoMsg.Type = ProtoMsgTyp_SetNodes
			protoMsg.ParentNodeId = n.id
			protoMsg.SetNodesWithLocalId(n.GetLocalConnIds(srcId))
			if len(protoMsg.Nodes) > 0 {
				if err = n.Reply(ctx, protoMsg); err != nil {
					if n.Logger != nil {
						n.Logger.Println("ProtoMsgTyp_QueryNodes err: CustomHandle enable node success reply err srcId", srcId, err)
					}
					return
				}
			}
			protoMsg.Type = ProtoMsgTyp_QueryNodes
			protoMsg.Nodes = nil
			if err = n.Broadcast(protoMsg, true, srcId, nil); err != nil {
				if n.Logger != nil {
					n.Logger.Println("ProtoMsgTyp_QueryNodes err: CustomHandle Broadcast err", err)
				}
				return
			}
		case ProtoMsgTyp_SetNodes:
			if len(protoMsg.Nodes) == 0 {
				return
			}
			for i := 0; i < len(protoMsg.Nodes); i++ {
				if protoMsg.Nodes[i].Id == n.id {
					continue
				}
				if protoMsg.IsAdd {
					n.Router.AddRoute(protoMsg.Nodes[i].Id, srcId, protoMsg.ParentNodeId, protoMsg.Nodes[i].Hop)
				} else {
					n.Router.DeleteRoute(protoMsg.Nodes[i].Id, srcId, protoMsg.ParentNodeId, protoMsg.Nodes[i].Hop)
				}
			}
			protoMsg.AddNop(1)
			if err = n.Broadcast(protoMsg, true, srcId, nil); err != nil {
				if n.Logger != nil {
					n.Logger.Println("ProtoMsgTyp_SetNodes err: CustomHandle Broadcast err", err)
				}
				return
			}
		default:
			if n.Logger != nil {
				n.Logger.Println("err: CustomHandle invalid protocol Message")
			}
		}
	}()
	return
}

func (n *NodeDiscoveryProtocol) AddNode(id uint32) bool {
	nod, exist := n.Find(id)

	if !exist || nod == nil || nod.Conn == nil {
		conn, ok := n.Get(id)
		if !ok {
			return false
		}
		n.Insert(conn)
		return true
	}
	return true
}

func (n *NodeDiscoveryProtocol) NewMsgWithConn(conn node.Conn, encoder Encoder) (*net.Message, error) {
	data, err := encoder.Encode()
	if err != nil {
		return nil, err
	}
	m := new(net.Message)
	m.Type = n.ProtocolMsgType
	m.SrcId = conn.LocalId()
	m.DestId = conn.RemoteId()
	m.Data = data
	return m, nil
}

type CustomReply interface {
	CustomReply(typ uint8, data []byte) error
}

func (n *NodeDiscoveryProtocol) Reply(r CustomReply, encoder Encoder) error {
	data, err := encoder.Encode()
	if err != nil {
		return err
	}
	return r.CustomReply(n.ProtocolMsgType, data)
}

func (n *NodeDiscoveryProtocol) Broadcast(en Encoder, enableFilter bool, filterId uint32, errFunc func(error) error) (err error) {
	var data []byte
	data, err = en.Encode()
	if err != nil {
		return err
	}
	m := new(net.Message)
	m.Type = n.ProtocolMsgType
	m.Data = data
	n.l.Lock()
	for id, info := range n.cache {
		if enableFilter && id == filterId {
			continue
		}
		m.SrcId = info.LocalId()
		m.DestId = info.RemoteId()
		if _, err = info.WriteMsg(m); err != nil {
			if errFunc != nil {
				if err = errFunc(err); err != nil {
					n.l.Unlock()
					return err
				}
			}
			if n.Logger.Writer() != nil {
				n.Logger.Printf("Broadcast err src %d, dst %d, err %v\n", m.SrcId, m.DestId, err)
			}
		}
	}
	n.l.Unlock()
	return nil
}

func (n *NodeDiscoveryProtocol) GetLocalConnIds(filter uint32) []uint32 {
	conns := n.Conns.GetAll()
	l := len(conns)
	result := make([]uint32, 0, l)
	for i := 0; i < l; i++ {
		if filter != conns[i].RemoteId() {
			result = append(result, conns[i].RemoteId())
		}
	}
	return result
}

func (n *NodeDiscoveryProtocol) Insert(conn node.Conn) {
	n.l.Lock()
	n.cache[conn.RemoteId()] = &UniteNode{
		UnixMill: time.Now().UnixMilli(),
		Conn:     conn,
	}
	n.l.Unlock()
}

func (n *NodeDiscoveryProtocol) Delete(id uint32) {
	n.l.Lock()
	delete(n.cache, id)
	n.l.Unlock()
}

func (n *NodeDiscoveryProtocol) Find(id uint32) (*UniteNode, bool) {
	n.l.Lock()
	pn, ok := n.cache[id]
	n.l.Unlock()
	return pn, ok
}

func (n *NodeDiscoveryProtocol) Disconnect(id uint32, err error) {
	node, ok := n.Find(id)
	if ok {
		_ = node.UnixMill
		n.Router.DeleteNextRoute(id)
		n.Delete(id)
	}

	protoMsg := NewProtoMsgWithOneNode(ProtoMsgTyp_SetNodes, false, id)
	protoMsg.ParentNodeId = n.id
	if err = n.Broadcast(protoMsg, false, 0, nil); err != nil {
		return
	}
}

type Encoder interface {
	Encode() ([]byte, error)
}

type UniteNode struct {
	UnixMill int64
	node.Conn
}

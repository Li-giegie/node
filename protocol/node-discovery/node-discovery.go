package node_discovery

import (
	"context"
	"github.com/Li-giegie/node/common"
	"log"
	"sync"
	"time"
)

type DiscoveryNode interface {
	Id() uint16
	GetConns() []common.Conn
	GetConn(id uint16) (common.Conn, bool)
	AddRoute(dst, next, hop, parentNode uint16)
	DeleteRoute(dst, next, hop, parentNode uint16) bool
	DeleteNextRoute(next uint16) bool
}

func NewNodeDiscoveryProtocol(c DiscoveryNode, ProtocolMsgType uint8) *NodeDiscoveryProtocol {
	return &NodeDiscoveryProtocol{
		DiscoveryNode:                   c,
		ProtocolMsgType:                 ProtocolMsgType,
		QueryEnableProtocolMaxNum:       3,
		QueryEnableProtocolIntervalStep: time.Millisecond * 500,
		l:                               new(sync.RWMutex),
		cache:                           make(map[uint16]*UniteNode),
	}
}

type NodeDiscoveryProtocol struct {
	DiscoveryNode
	ProtocolMsgType                 uint8
	QueryEnableProtocolMaxNum       int
	QueryEnableProtocolIntervalStep time.Duration
	l                               *sync.RWMutex
	cache                           map[uint16]*UniteNode
}

func (n *NodeDiscoveryProtocol) StartTimingQueryEnableProtoNode(ctx context.Context, timeout time.Duration) (err error) {
	ok := true
	stopChan := make(chan struct{})
	go func() {
		protoMsg := NewProtoMsgWithType(ProtoMsgTyp_QueryEnable)
		for ok {
			time.Sleep(timeout)
			if err = n.Broadcast(protoMsg, false, 0, nil); err != nil {
				log.Println("NodeDiscoveryProtocol.StartTimingQueryEnableProtoNode.Broadcast.ProtoMsgTyp_QueryEnable err", err)
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

func (n *NodeDiscoveryProtocol) Connection(conn common.Conn) {
	go func() {
		dstId := conn.RemoteId()
		protoMsg := NewProtoMsgWithOneNode(ProtoMsgTyp_SetNodes, true, dstId)
		protoMsg.ParentNodeId = n.Id()
		err := n.Broadcast(protoMsg, true, dstId, nil)
		if err != nil {
			log.Println("NodeDiscoveryProtocol.Connection err", err)
			return
		}
		protoMsg.Type = ProtoMsgTyp_QueryEnable
		protoMsg.Nodes = nil
		msg, err := n.NewMsgWithConn(conn, protoMsg)
		if err != nil {
			log.Println("NodeDiscoveryProtocol.Connection NewMsgWithConn err", err)
			return
		}
		for i := 1; i <= n.QueryEnableProtocolMaxNum; i++ {
			if err = conn.WriteMsg(msg); err != nil {
				log.Println("NodeDiscoveryProtocol.Connection WriteMsg err", err)
				return
			}
			time.Sleep(n.QueryEnableProtocolIntervalStep * time.Duration(i))
			if _, ok := n.Find(dstId); ok {
				return
			}
		}
	}()
}

func (n *NodeDiscoveryProtocol) CustomHandle(ctx common.Context) (next bool) {
	if n.ProtocolMsgType != ctx.Type() {
		return true
	}
	next = false
	go func() {
		srcId := ctx.SrcId()
		protoMsg, err := new(ProtoMsg).Decode(ctx.Data())
		if err != nil {
			log.Println("NodeDiscoveryProtocol.CustomHandle err", err)
			return
		}
		switch protoMsg.Type {
		case ProtoMsgTyp_QueryEnable:
			if !n.AddNode(srcId) {
				log.Println("NodeDiscoveryProtocol.CustomHandle.ProtoMsgTyp_QueryEnable err", err)
				return
			}
			protoMsg.Type = ProtoMsgTyp_ResponseEnable
			if err = n.Reply(ctx, protoMsg); err != nil {
				log.Println("NodeDiscoveryProtocol.CustomHandle.NodeDiscoveryMsgType_QueryProtocol Reply err", err)
			}
		case ProtoMsgTyp_ResponseEnable:
			if !n.AddNode(srcId) {
				log.Println("NodeDiscoveryProtocol.CustomHandle.ProtoMsgTyp_ResponseEnable err", err)
				return
			}
			protoMsg.Type = ProtoMsgTyp_QueryNodes
			if err = n.Reply(ctx, protoMsg); err != nil {
				log.Println("NodeDiscoveryProtocol.CustomHandle.ProtoMsgTyp_ResponseEnable Reply err", err)
			}
		case ProtoMsgTyp_QueryNodes:
			protoMsg.IsAdd = true
			protoMsg.Type = ProtoMsgTyp_SetNodes
			protoMsg.ParentNodeId = n.Id()
			protoMsg.SetNodesWithLocalId(n.GetLocalConnIds(srcId))
			if len(protoMsg.Nodes) > 0 {
				if err = n.Reply(ctx, protoMsg); err != nil {
					log.Println("NodeDiscoveryProtocol.CustomHandle.ProtoMsgTyp_QueryNodes Reply err", err)
					return
				}
			}
			protoMsg.Type = ProtoMsgTyp_QueryNodes
			protoMsg.Nodes = nil
			if err = n.Broadcast(protoMsg, true, srcId, nil); err != nil {
				log.Println("NodeDiscoveryProtocol.CustomHandle.ProtoMsgTyp_QueryNodes Broadcast err", err)
				return
			}
		case ProtoMsgTyp_SetNodes:
			if len(protoMsg.Nodes) == 0 {
				return
			}
			for i := 0; i < len(protoMsg.Nodes); i++ {
				if protoMsg.Nodes[i].Id == n.Id() {
					continue
				}
				if protoMsg.IsAdd {
					n.AddRoute(protoMsg.Nodes[i].Id, srcId, protoMsg.Nodes[i].Hop, protoMsg.ParentNodeId)
				} else {
					n.DeleteRoute(protoMsg.Nodes[i].Id, srcId, protoMsg.Nodes[i].Hop, protoMsg.ParentNodeId)
				}
			}
			protoMsg.AddNop(1)
			if err = n.Broadcast(protoMsg, true, srcId, nil); err != nil {
				log.Println("NodeDiscoveryProtocol.CustomHandle.ProtoMsgTyp_SetNodes Broadcast err", err)
				return
			}
		default:
			log.Println("invalid protocol Message")
		}
	}()
	return
}

func (n *NodeDiscoveryProtocol) AddNode(id uint16) bool {
	nod, exist := n.Find(id)
	if !exist || nod == nil || nod.Conn == nil || nod.Conn.State() != common.ConnStateTypeOnConnect {
		conn, ok := n.GetConn(id)
		if !ok {
			return false
		}
		n.Insert(conn)
		return true
	}
	return true
}

func (n *NodeDiscoveryProtocol) NewMsgWithConn(conn common.Conn, encoder Encoder) (*common.Message, error) {
	data, err := encoder.Encode()
	if err != nil {
		return nil, err
	}
	m := new(common.Message)
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

func (n *NodeDiscoveryProtocol) Broadcast(en Encoder, enableFilter bool, filterId uint16, errFunc func(error) error) (err error) {
	var data []byte
	data, err = en.Encode()
	if err != nil {
		return err
	}
	m := new(common.Message)
	m.Type = n.ProtocolMsgType
	m.Data = data
	n.l.Lock()
	for id, info := range n.cache {
		if enableFilter && id == filterId {
			continue
		}
		m.SrcId = info.LocalId()
		m.DestId = info.RemoteId()
		if err = info.WriteMsg(m); err != nil {
			if errFunc != nil {
				if err = errFunc(err); err != nil {
					n.l.Unlock()
					return err
				}
			}
			log.Printf("ProtocolNodes.Broadcast err src %d, dst %d, err %v\n", m.SrcId, m.DestId, err)
		}
	}
	n.l.Unlock()
	return nil
}

func (n *NodeDiscoveryProtocol) GetLocalConnIds(filter uint16) []uint16 {
	conns := n.GetConns()
	l := len(conns)
	result := make([]uint16, 0, l)
	for i := 0; i < l; i++ {
		if filter != conns[i].RemoteId() {
			result = append(result, conns[i].RemoteId())
		}
	}
	return result
}

func (n *NodeDiscoveryProtocol) Insert(conn common.Conn) {
	n.l.Lock()
	n.cache[conn.RemoteId()] = &UniteNode{
		UnixMill: time.Now().UnixMilli(),
		Conn:     conn,
	}
	n.l.Unlock()
}

func (n *NodeDiscoveryProtocol) Delete(id uint16) {
	n.l.Lock()
	delete(n.cache, id)
	n.l.Unlock()
}

func (n *NodeDiscoveryProtocol) Find(id uint16) (*UniteNode, bool) {
	n.l.Lock()
	pn, ok := n.cache[id]
	n.l.Unlock()
	return pn, ok
}

func (n *NodeDiscoveryProtocol) Disconnect(id uint16, err error) {
	node, ok := n.Find(id)
	if ok {
		_ = node.UnixMill
		n.DeleteNextRoute(id)
		n.Delete(id)
	}

	protoMsg := NewProtoMsgWithOneNode(ProtoMsgTyp_SetNodes, false, id)
	protoMsg.ParentNodeId = n.Id()
	if err = n.Broadcast(protoMsg, false, 0, nil); err != nil {
		log.Println("Disconnect err", err)
		return
	}
}

type Encoder interface {
	Encode() ([]byte, error)
}

type UniteNode struct {
	UnixMill int64
	common.Conn
}

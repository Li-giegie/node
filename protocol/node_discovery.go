package protocol

import (
	"encoding/json"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"log"
	"sync"
	"time"
)

var (
	NodeDiscoveryMsgType_QueryProtocol = GetMsgType()
	NodeDiscoveryMsgType_ReplyProtocol = GetMsgType()
	NodeDiscoveryMsgType_QueryNodes    = GetMsgType()
	NodeDiscoveryMsgType_ReplyNodes    = GetMsgType()
)

type NodeDiscoveryProtocol struct {
	node.Server
	queryInterval time.Duration
	rootNode      map[uint16]*rootNode
	rootNodeLock  *sync.RWMutex
}

type rootNode struct {
	syncUnixMill int64
}

func NewNodeDiscoveryProtocol() *NodeDiscoveryProtocol {
	return &NodeDiscoveryProtocol{rootNode: map[uint16]*rootNode{}, rootNodeLock: &sync.RWMutex{}, queryInterval: time.Millisecond * 500}
}

// InitServer interval 间隔时间后，同步节点数据
func (n *NodeDiscoveryProtocol) InitServer(server node.Server, interval time.Duration) {
	n.Server = server
	log.Println(NodeDiscoveryMsgType_QueryProtocol)
	log.Println(NodeDiscoveryMsgType_ReplyProtocol)
	log.Println(NodeDiscoveryMsgType_QueryNodes)
	log.Println(NodeDiscoveryMsgType_ReplyNodes)
	//for {
	//	time.Sleep(interval)
	//	for _, conn := range n.Server.GetConns() {
	//		_ = conn.WriteMsg(&common.Message{
	//			Type:   NodeDiscoveryMsgType_QueryProtocol,
	//			SrcId:  conn.LocalId(),
	//			DestId: conn.RemoteId(),
	//		})
	//	}
	//}
}

// Connection 查询节点是否启用节点发现协议，最多发送3次查询消息，递增时长，在3次后仍然没有收到回复，那么认为该节点没有开启协议
func (n *NodeDiscoveryProtocol) Connection(conn common.Conn) {
	go func() {
		for i := 1; i <= 3; i++ {
			err := conn.WriteMsg(&common.Message{
				Type:   NodeDiscoveryMsgType_QueryProtocol,
				SrcId:  conn.LocalId(),
				DestId: conn.RemoteId(),
			})
			if err != nil {
				log.Println("查询消息发送失败", err)
				return
			}
			time.Sleep(n.queryInterval * time.Duration(i))
			n.rootNodeLock.RLock()
			_, ok := n.rootNode[conn.RemoteId()]
			n.rootNodeLock.RUnlock()
			if ok {
				return
			}
		}
		n.rootNodeLock.Lock()
		defer n.rootNodeLock.Unlock()
		info := new(nodeInfo)
		info.Type = nodeInfoType_Add
		info.Nodes = []uint16{conn.RemoteId()}
		data, err := json.Marshal(info)
		if err != nil {
			log.Println("Connection err", err)
			return
		}
		for u, r := range n.rootNode {
			c, ok := n.Server.GetConn(u)
			if !ok {
				continue
			}
			err = c.WriteMsg(&common.Message{
				Type:   NodeDiscoveryMsgType_ReplyNodes,
				SrcId:  c.LocalId(),
				DestId: c.RemoteId(),
				Data:   data,
			})
			if err != nil {
				log.Println("Connection err -0", err)
				continue
			}
			r.syncUnixMill = time.Now().UnixMilli()
		}
	}()
}

type nodeInfoType uint16

const (
	nodeInfoType_Delete = iota
	nodeInfoType_Add
)

type nodeInfo struct {
	Type  uint16
	Nodes []uint16
}

func (n *NodeDiscoveryProtocol) CustomHandle(ctx common.Context) (next bool) {
	if ctx.Type() != NodeDiscoveryMsgType_QueryProtocol && ctx.Type() != NodeDiscoveryMsgType_ReplyProtocol &&
		ctx.Type() != NodeDiscoveryMsgType_QueryNodes && ctx.Type() != NodeDiscoveryMsgType_ReplyNodes {
		return true
	}
	go func() {
		switch ctx.Type() {
		case NodeDiscoveryMsgType_QueryProtocol:
			src := ctx.SrcId()
			if err := ctx.CustomReply(NodeDiscoveryMsgType_ReplyProtocol, nil); err == nil {
				n.rootNodeLock.Lock()
				_, ok := n.rootNode[src]
				if !ok {
					n.rootNode[src] = new(rootNode)
				}
				n.rootNodeLock.Unlock()
				log.Println("添加root节点", src)
			}
		case NodeDiscoveryMsgType_ReplyProtocol:
			n.rootNodeLock.Lock()
			if _, ok := n.rootNode[ctx.SrcId()]; !ok {
				n.rootNode[ctx.SrcId()] = new(rootNode)
			}
			n.rootNodeLock.Unlock()
			if err := ctx.CustomReply(NodeDiscoveryMsgType_QueryNodes, nil); err != nil {
				log.Println("CustomReply NodeDiscoveryMsgType_ReplyProtocol err", err)
			}
			log.Println("添加root节点", ctx.DestId())
		case NodeDiscoveryMsgType_QueryNodes:
			log.Println("查询root节点", ctx.SrcId())
			info := new(nodeInfo)
			info.Type = 1
			info.Nodes = n.GetLocalConnIds()
			data, err := json.Marshal(info)
			if err != nil {
				log.Println("CustomReply NodeDiscoveryMsgType_QueryNodes err", err)
				return
			}
			if err = ctx.CustomReply(NodeDiscoveryMsgType_ReplyNodes, data); err != nil {
				log.Println("CustomReply NodeDiscoveryMsgType_QueryNodes err -0", err)
				return
			}
			n.rootNodeLock.Lock()
			defer n.rootNodeLock.Unlock()
			for id, r := range n.rootNode {
				if id == ctx.DestId() {
					continue
				}
				conn, ok := n.Server.GetConn(id)
				if !ok {
					continue
				}
				err = conn.WriteMsg(&common.Message{
					Type:   NodeDiscoveryMsgType_QueryNodes,
					SrcId:  conn.LocalId(),
					DestId: conn.RemoteId(),
				})
				if err != nil {
					log.Println("CustomReply NodeDiscoveryMsgType_QueryNodes err -1", err)
					return
				}
				r.syncUnixMill = time.Now().UnixMilli()
			}
		case NodeDiscoveryMsgType_ReplyNodes:
			log.Println("响应root节点", ctx.SrcId())
			info := new(nodeInfo)
			err := json.Unmarshal(ctx.Data(), info)
			if err != nil {
				log.Println("CustomReply NodeDiscoveryMsgType_ReplyNodes err", err)
				return
			}
			if len(info.Nodes) == 0 {
				return
			}
			for i := 0; i < len(info.Nodes); i++ {
				if info.Nodes[i] == n.Server.Id() {
					continue
				}
				switch info.Type {
				case nodeInfoType_Delete:
					n.DeleteRouteNextHop(info.Nodes[i], ctx.SrcId(), 0)
					log.Println(ctx.DestId(), "删除一条路由", info.Nodes[i], ctx.SrcId())
				case nodeInfoType_Add:
					n.AddRoute(info.Nodes[i], ctx.SrcId(), 0)
					log.Println(ctx.DestId(), "添加一条路由", info.Nodes[i], ctx.SrcId())
				}

			}
			n.rootNodeLock.Lock()
			defer n.rootNodeLock.Unlock()
			for u, r := range n.rootNode {
				log.Println("转发给", u)
				if u == ctx.SrcId() {
					continue
				}
				conn, ok := n.Server.GetConn(u)
				if !ok {
					continue
				}
				err = conn.WriteMsg(&common.Message{
					Type:   NodeDiscoveryMsgType_ReplyNodes,
					SrcId:  conn.LocalId(),
					DestId: conn.RemoteId(),
					Data:   ctx.Data(),
				})
				if err != nil {
					log.Println("CustomReply NodeDiscoveryMsgType_ReplyNodes err -0", err)
					return
				}
				r.syncUnixMill = time.Now().UnixMilli()
			}
		}
	}()
	return false
}

func (n *NodeDiscoveryProtocol) Disconnect(id uint16, err error) {
	rootIds := make([]uint16, 0, len(n.rootNode))
	n.rootNodeLock.Lock()
	delete(n.rootNode, id)
	for u, _ := range n.rootNode {
		rootIds = append(rootIds, u)
	}
	n.rootNodeLock.Unlock()
	info := new(nodeInfo)
	info.Type = nodeInfoType_Delete
	info.Nodes = []uint16{id}
	data, err := json.Marshal(info)
	if err != nil {
		log.Println("Disconnect err", err)
		return
	}
	for _, rootId := range rootIds {
		conn, ok := n.Server.GetConn(rootId)
		if !ok {
			continue
		}
		_ = conn.WriteMsg(&common.Message{
			Type:   NodeDiscoveryMsgType_ReplyNodes,
			SrcId:  conn.LocalId(),
			DestId: conn.RemoteId(),
			Data:   data,
		})
	}

}

func (n *NodeDiscoveryProtocol) GetLocalConnIds() []uint16 {
	conns := n.GetConns()
	l := len(conns)
	result := make([]uint16, 0, l)
	for i := 0; i < l; i++ {
		result = append(result, conns[i].RemoteId())
	}
	return result
}

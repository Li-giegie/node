package nodediscovery

import (
	"encoding/json"
	"fmt"
)

const (
	ProtoMsgTyp_QueryEnable uint8 = 1 + iota
	ProtoMsgTyp_ResponseEnable
	ProtoMsgTyp_QueryNodes
	ProtoMsgTyp_SetNodes
)

func NewProtoMsg(pt uint8, isAdd bool, n []*Node) *ProtoMsg {
	return &ProtoMsg{
		Type:  pt,
		IsAdd: isAdd,
		Nodes: n,
	}
}
func NewProtoMsgWithOneNode(pt uint8, isAdd bool, id uint32) *ProtoMsg {
	return &ProtoMsg{
		Type:  pt,
		IsAdd: isAdd,
		Nodes: []*Node{{Id: id, Hop: 1}},
	}
}
func NewProtoMsgWithType(pt uint8) *ProtoMsg {
	return &ProtoMsg{
		Type: pt,
	}
}
func NewProtoMsgWithIds(pt uint8, isAdd bool, id []uint32) *ProtoMsg {
	pm := new(ProtoMsg)
	pm.Type = pt
	pm.IsAdd = isAdd
	pm.Nodes = make([]*Node, len(id))
	for i := 0; i < len(id); i++ {
		pm.Nodes[i] = &Node{Id: id[i], Hop: 1}
	}
	return pm
}

type ProtoMsg struct {
	Type         uint8
	IsAdd        bool
	ParentNodeId uint32
	Nodes        []*Node
}

func (p *ProtoMsg) String() string {
	data, _ := json.MarshalIndent(p, "", "\t")
	return string(data)
}

type Node struct {
	Id  uint32
	Hop uint16
}

func (n *Node) String() string {
	return fmt.Sprintf("id: %d, hop: %d", n.Id, n.Hop)
}

func (p *ProtoMsg) SetNodesWithLocalId(id []uint32) {
	p.Nodes = make([]*Node, len(id))
	for i := 0; i < len(id); i++ {
		p.Nodes[i] = &Node{Id: id[i], Hop: 1}
	}
}

func (p *ProtoMsg) AddNop(step uint16) {
	for i := 0; i < len(p.Nodes); i++ {
		p.Nodes[i].Hop += step
	}
}

func (p *ProtoMsg) Encode() ([]byte, error) {
	return json.Marshal(p)
}

func (p *ProtoMsg) Decode(data []byte) (*ProtoMsg, error) {
	err := json.Unmarshal(data, p)
	return p, err
}

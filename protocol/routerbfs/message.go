package routerbfs

import "encoding/json"

const (
	Action_AddNode uint8 = iota
	Action_RemoveNode
	Action_Query
	Action_Reply
)

type NodeInfo struct {
	RootNodeId  uint32
	SubNodeInfo []*SubInfo
}

type SubInfo struct {
	Id      uint32
	UnixNao int64
}

type ProtoMsg struct {
	Id     uint32
	SrcId  uint32
	Action uint8
	NInfo  []*NodeInfo
}

func (m *ProtoMsg) Encode() []byte {
	data, _ := json.Marshal(m)
	return data
}

func (m *ProtoMsg) Decode(b []byte) error {
	return json.Unmarshal(b, m)
}

func (m *ProtoMsg) String() string {
	data, _ := json.Marshal(m)
	return string(data)
}

package routerbfsv2

import (
	"encoding/json"
	"github.com/Li-giegie/node/pkg/router"
)

const (
	Action_Open uint8 = iota
	Action_Refresh
	Action_AddRoutes
	Action_RemoveRoutes
)

func DecodeProtoMsg(data []byte) (result *ProtoMsg, err error) {
	result = new(ProtoMsg)
	err = json.Unmarshal(data, &result)
	return
}

type ProtoMsg struct {
	Action   uint8
	SrcId    uint32
	Paths    []uint32
	UnixNano int64
	Routes   []*router.RouteEmpty `json:"routes,omitempty"`
	Nodes    []uint32             `json:"nodes,omitempty"`
}

func (m *ProtoMsg) Encode() []byte {
	data, _ := json.Marshal(m)
	return data
}

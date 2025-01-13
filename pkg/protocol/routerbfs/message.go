package routerbfs

import (
	"encoding/json"
	"fmt"
	"github.com/Li-giegie/node/pkg/errors"
)

const (
	Action_Unknown = iota
	Action_AddNode
	Action_RemoveNode
	Action_QueryProtocol
	Action_ReplyProtocol
	Action_SyncHash
	Action_SyncQueryNode
	Action_SyncReplyNode
	Action_Undefine
)

var errInvalidAction = errors.New("invalid action")

func decodeProtoMsg(b []byte) (*ProtoMsg, error) {
	p := new(ProtoMsg)
	err := p.Decode(b)
	if err != nil {
		return nil, err
	}
	if p.Action < Action_Unknown || p.Action > Action_Undefine {
		return nil, errInvalidAction
	}
	return p, nil
}

type ProtoMsg struct {
	Action   uint8
	Paths    []uint32   `json:"paths,omitempty"`
	Nodes    []NodeInfo `json:"nodes,omitempty"`
	SyncInfo *SyncInfo  `json:"synInfo,omitempty"`
}

func (m *ProtoMsg) Encode() []byte {
	p, _ := json.Marshal(m)
	return p
}

func (m *ProtoMsg) Decode(p []byte) error {
	return json.Unmarshal(p, m)
}

func (m *ProtoMsg) String() string {
	switch m.Action {
	case Action_QueryProtocol:
		return fmt.Sprintf("Action_QueryProtocol paths %v", m.Paths)
	case Action_ReplyProtocol:
		return fmt.Sprintf("Action_ReplyProtocol paths %v nodes %+v", m.Paths, m.Nodes)
	case Action_AddNode:
		return fmt.Sprintf("Action_AddNode paths %v nodes %+v", m.Paths, m.Nodes)
	case Action_RemoveNode:
		return fmt.Sprintf("Action_RemoveNode paths %v nodes %+v", m.Paths, m.Nodes)
	case Action_SyncHash:
		return fmt.Sprintf("Action_SyncNode paths %+v syncnode %+v", m.Paths, m.SyncInfo)
	case Action_SyncQueryNode:
		return fmt.Sprintf("Action_SyncNode paths %+v syncnode %+v", m.Paths, m.SyncInfo)
	case Action_SyncReplyNode:
		return fmt.Sprintf("Action_SyncNode paths %+v syncnode %+v", m.Paths, m.SyncInfo)
	default:
		return fmt.Sprintf("invalid action %v %v %v %v", m.Action, m.Paths, m.Nodes, m.SyncInfo)
	}
}

type NodeInfo struct {
	RootId uint32
	SubIds []SubInfo
}

type SubInfo struct {
	Id      uint32
	UnixNao int64
}

type SyncInfo struct {
	Hash             uint64
	NodeNum          uint32
	ValidityUnixNano int64
	*SyncNode
}

type SyncNode struct {
	RootId uint32
	SubIds []uint32
}

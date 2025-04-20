package routerbfs

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/errors"
)

type Action uint8

const (
	Action_NeighborASK Action = 1 + iota
	Action_NeighborACK
	Action_PullNode
	Action_PushNode
	Action_Update
	Action_SyncHash
	Action_SyncQueryNode
	Action_SyncNode
)

func (action Action) String() string {
	switch action {
	case Action_Update:
		return "Update"
	case Action_NeighborASK:
		return "NeighborASK"
	case Action_NeighborACK:
		return "NeighborACK"
	case Action_SyncHash:
		return "SyncHash"
	case Action_SyncQueryNode:
		return "SyncQueryNode"
	case Action_PullNode:
		return "PullNode"
	case Action_PushNode:
		return "PushNode"
	case Action_SyncNode:
		return "SyncNode"
	default:
		return "Unknown"
	}
}

type ProtoMsg struct {
	Id     int64
	Action Action
	SrcId  uint32
	Paths  []uint32
	Data   []byte
}

func (m *ProtoMsg) Encode() []byte {
	buf := make([]byte, 17+(len(m.Paths)*4)+len(m.Data))
	buf[0] = byte(m.Action)
	binary.LittleEndian.PutUint64(buf[1:9], uint64(m.Id))
	binary.LittleEndian.PutUint32(buf[9:13], m.SrcId)
	binary.LittleEndian.PutUint32(buf[13:17], uint32(len(m.Paths)))
	index := 17
	for _, path := range m.Paths {
		binary.LittleEndian.PutUint32(buf[index:], path)
		index += 4
	}
	copy(buf[index:], m.Data)
	return buf
}

func (m *ProtoMsg) Decode(p []byte) error {
	if len(p) < 17 {
		return errors.New("message too short")
	}
	m.Action = Action(p[0])
	m.Id = int64(binary.LittleEndian.Uint64(p[1:9]))
	m.SrcId = binary.LittleEndian.Uint32(p[9:13])
	l := int(binary.LittleEndian.Uint32(p[13:17]))
	m.Paths = make([]uint32, l)
	index := 17
	for i := 0; i < l; i++ {
		m.Paths[i] = binary.LittleEndian.Uint32(p[index:])
		index += 4
	}
	m.Data = p[index:]
	return nil
}

func (m *ProtoMsg) Valid() error {
	if len(m.Paths) == 0 {
		return errors.New("paths is invalid")
	}
	if m.Action < 1 || m.Action > 8 {
		return errors.New("action is invalid")
	}
	return nil
}

func (m *ProtoMsg) String() string {
	return fmt.Sprintf("SrcId: %d, Action: %s, Id: %d, Paths: %v, Data: %s", m.SrcId, m.Action.String(), m.Id, m.Paths, m.Data)
}

type SyncMsg struct {
	Id         uint32
	SubNodeNum uint32
	Hash       uint32
}

func (m *SyncMsg) Encode() []byte {
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint32(buf[:4], m.Id)
	binary.LittleEndian.PutUint32(buf[4:8], m.SubNodeNum)
	binary.LittleEndian.PutUint32(buf[8:12], m.Hash)
	return buf
}

func (m *SyncMsg) Decode(b []byte) error {
	if len(b) != 16 {
		return errors.New("invalid SyncMsg length")
	}
	m.Id = binary.LittleEndian.Uint32(b[:4])
	m.SubNodeNum = binary.LittleEndian.Uint32(b[4:8])
	m.Hash = binary.LittleEndian.Uint32(b[8:12])
	return nil
}

type UpdateAction uint8

func (u UpdateAction) String() string {
	switch u {
	case UpdateAction_AddNode:
		return "AddNode"
	case UpdateAction_RemoveNode:
		return "RemoveRoot"
	}
	return "invalid UpdateAction"
}

const (
	UpdateAction_AddNode UpdateAction = 1 + iota
	UpdateAction_RemoveNode
)

type UpdateMsg []*UpdateMsgEntry

func (u *UpdateMsg) Encode() []byte {
	data, _ := json.Marshal(u)
	return data
}

func (u *UpdateMsg) Decode(data []byte) error {
	return json.Unmarshal(data, u)
}

type UpdateMsgEntry struct {
	Action  UpdateAction
	RootId  uint32
	SubId   uint32
	SubType conn.NodeType
}

type List []uint32

func (l *List) Encode() []byte {
	data, _ := json.Marshal(l)
	return data
}

func (l *List) Decode(data []byte) error {
	return json.Unmarshal(data, l)
}

func (l *List) Map() map[uint32]struct{} {
	m := make(map[uint32]struct{}, len(*l))
	for _, u := range *l {
		m[u] = struct{}{}
	}
	return m
}

type PushMsg []PushMsgEntry

func (p *PushMsg) Encode() []byte {
	data, _ := json.Marshal(p)
	return data
}

func (p *PushMsg) Decode(data []byte) error {
	return json.Unmarshal(data, p)
}

type PushMsgEntry struct {
	Root     uint32
	SubEntry []PushMsgSubEntry
}

func (p *PushMsgEntry) Encode() []byte {
	data, _ := json.Marshal(p)
	return data
}

func (p *PushMsgEntry) Decode(data []byte) error {
	return json.Unmarshal(data, p)
}

type PushMsgSubEntry struct {
	SubId   uint32
	SubType conn.NodeType
}

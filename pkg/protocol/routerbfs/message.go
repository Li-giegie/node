package routerbfs

import (
	"encoding/binary"
	"fmt"
	"github.com/Li-giegie/node/pkg/errors"
)

type Action uint8

const (
	Action_NeighborASK Action = 1 + iota
	Action_NeighborACK
	Action_Update
	Action_SyncHash
	Action_SyncQueryNode
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
	default:
		return "Unknown"
	}
}

func decodeProtoMsg(b []byte) (*ProtoMsg, error) {
	p := new(ProtoMsg)
	err := p.Decode(b)
	return p, err
}

type ProtoMsg struct {
	Action   Action
	UnixNano int64
	SrcId    uint32
	Paths    []uint32
	Data     []byte
}

func (m *ProtoMsg) Encode() []byte {
	buf := make([]byte, 17+(len(m.Paths)*4)+len(m.Data))
	buf[0] = byte(m.Action)
	binary.LittleEndian.PutUint64(buf[1:9], uint64(m.UnixNano))
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

func (m *ProtoMsg) Valid() error {
	if len(m.Paths) == 0 {
		return errors.New("paths is invalid")
	}
	if m.UnixNano <= 0 {
		return errors.New("unixNano is invalid")
	}
	if m.Action < 1 || m.Action > 5 {
		return errors.New("action is invalid")
	}
	return nil
}

func (m *ProtoMsg) Decode(p []byte) error {
	if len(p) < 17 {
		return errors.New("message too short")
	}
	m.Action = Action(p[0])
	m.UnixNano = int64(binary.LittleEndian.Uint64(p[1:9]))
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

func (m *ProtoMsg) String() string {
	return fmt.Sprintf("Action: %s, UnixNano: %d, SrcId: %d, Paths: %v, Data: %s", m.Action.String(), m.UnixNano, m.SrcId, m.Paths, m.Data)
}

type SyncMsg struct {
	Id         uint32
	SubNodeNum uint32
	Hash       uint64
}

type UpdateAction uint8

func (u UpdateAction) String() string {
	switch u {
	case UpdateAction_AddRoot:
		return "AddRoot"
	case UpdateAction_RemoveRoot:
		return "RemoveRoot"
	case UpdateAction_AddSub:
		return "AddSub"
	case UpdateAction_DeleteSub:
		return "DeleteSub"
	}
	return "invalid UpdateAction"
}

const (
	UpdateAction_AddSub UpdateAction = iota
	UpdateAction_DeleteSub
	UpdateAction_AddRoot
	UpdateAction_RemoveRoot
)

type UpdateMsg struct {
	Action UpdateAction
	RootId uint32
	SubId  uint32
}

package message

import "fmt"

// 标准消息类型
const (
	MsgType_Default uint8 = iota
	MsgType_Reply
	MsgType_ReplyErr
	MsgType_Undefined
)

const MsgHeaderLen = 1 + 1 + 4 + 4 + 4 + 4 + 2

type Message struct {
	Type   uint8  //消息类型，用于特定功能（协议）而不是不同场景，不可滥用，Data字段能解决所有场景
	Hop    uint8  //消息的跳数
	Id     uint32 //消息唯一标识
	SrcId  uint32 //源节点
	DestId uint32 //目的节点
	Data   []byte //消息内容
}

func (m *Message) String() string {
	return fmt.Sprintf("type: %d, id: %v, srcId: %v, destId: %v, hop: %d, data: %s", m.Type, m.Id, m.SrcId, m.DestId, m.Hop, m.Data)
}

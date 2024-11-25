package message

import "fmt"

// 标准消息类型
const (
	MsgType_Send uint8 = iota
	MsgType_Reply
	MsgType_ReplyErrConnNotExist
	MsgType_ReplyErrLenLimit
	MsgType_ReplyErrCheckSum
	MsgType_ReplyErr
	Null
)

const MsgHeaderLen = 1 + 1 + 4 + 4 + 4 + 4 + 2

type Message struct {
	Type   uint8  //消息类型，不同的协议该值不同
	Hop    uint8  //消息的跳数
	Id     uint32 //消息唯一标识
	SrcId  uint32 //源节点
	DestId uint32 //目的节点
	Data   []byte //消息内容
}

func (m *Message) String() string {
	return fmt.Sprintf("type: %d, id: %v, srcId: %v, destId: %v, hop: %d, data: %s", m.Type, m.Id, m.SrcId, m.DestId, m.Hop, m.Data)
}

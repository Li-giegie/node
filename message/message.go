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

const MsgHeaderLen = 1 + 4 + 4 + 4 + 4 + 2

type Message struct {
	Type   uint8
	Id     uint32
	SrcId  uint32
	DestId uint32
	Data   []byte
}

func (m *Message) String() string {
	return fmt.Sprintf("Message { type: %d, id: %v, srcId: %v, destId: %v, data: %s}", m.Type, m.Id, m.SrcId, m.DestId, m.Data)
}

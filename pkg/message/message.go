package message

import "fmt"

// 标准消息类型
const (
	MsgType_Default uint8 = iota
	MsgType_Reply
	MsgType_KeepaliveASK
	MsgType_KeepaliveACK
	MsgType_Undefined
)

const (
	StateCode_CheckSumInvalid int16 = 100 + iota
	StateCode_RequestTimeout
	StateCode_LengthOverflow
	StateCode_Success            int16 = 200
	StateCode_ResponseInvalid    int16 = 204
	StateCode_NodeNotExist       int16 = 404
	StateCode_MessageTypeInvalid int16 = 600
)

const MsgHeaderLen = 1 + 1 + 4 + 4 + 4 + 4 + 2

type Message struct {
	Type   uint8  //消息类型，用于特定功能（协议）而不是不同场景，不可滥用，Data字段能解决所有场景
	Hop    uint8  //消息的跳数，初始值0，每经过一个节点加1
	Id     uint32 //消息唯一标识，请求时（Request系列方法）必须唯一，每个请求如果有相应都对应一个唯一的响应，发送时该字段可以忽略
	SrcId  uint32 //源节点
	DestId uint32 //目的节点
	Data   []byte //消息数据
}

func (m *Message) String() string {
	return fmt.Sprintf("type: %d, id: %v, srcId: %v, destId: %v, hop: %d, data: %s", m.Type, m.Id, m.SrcId, m.DestId, m.Hop, m.Data)
}

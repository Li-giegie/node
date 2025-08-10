package reply

import (
	"encoding/json"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/errors"
	"github.com/Li-giegie/node/pkg/message"
)

func NewReply(conn *conn.Conn, mId, dstId uint32) *Reply {
	return &Reply{
		conn:     conn,
		msgId:    mId,
		msgDstId: dstId,
	}
}

type Reply struct {
	conn     *conn.Conn
	msgId    uint32
	msgDstId uint32
	response bool
}

// Write 回复数据，type为 message.MsgType_Reply，限制回复一次，不要尝试多次回复，多次回复返回 var ErrLimitReply = errors.New("limit reply to one time")
func (c *Reply) Write(code int16, data []byte) error {
	if c.response {
		return errors.ErrMultipleResponse
	}
	c.response = true
	reData := make([]byte, 2+len(data))
	reData[0], reData[1] = byte(code), byte(code>>8)
	copy(reData[2:], data)
	return c.conn.SendMessage(&message.Message{
		Type:   message.MsgType_Response,
		Hop:    0,
		Id:     c.msgId,
		SrcId:  c.conn.LocalId(),
		DestId: c.msgDstId,
		Data:   reData,
	})
}

func (c *Reply) String(code int16, data string) error {
	return c.Write(code, []byte(data))
}

func (c *Reply) JSON(code int16, data any) error {
	p, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.Write(code, p)
}

func (c *Reply) GetConn() *conn.Conn {
	return c.conn
}

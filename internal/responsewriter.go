package internal

import (
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/conn/implconn"
	"github.com/Li-giegie/node/pkg/errors"
	"github.com/Li-giegie/node/pkg/message"
)

type ResponseWriter struct {
	*implconn.Conn
	MsgId    uint32
	MsgDstId uint32
	response bool
}

// Response 响应数据，type为 message.MsgType_Reply，限制回复一次，不要尝试多次回复，多次回复返回 var ErrLimitReply = errors.New("limit reply to one time")
func (c *ResponseWriter) Response(code int16, data []byte) error {
	if c.response {
		return errors.ErrMultipleResponse
	}
	c.response = true
	reData := make([]byte, 2+len(data))
	reData[0], reData[1] = byte(code), byte(code>>8)
	copy(reData[2:], data)
	return c.SendMessage(&message.Message{
		Type:   message.MsgType_Response,
		Hop:    0,
		Id:     c.MsgId,
		SrcId:  c.Conn.LocalId(),
		DestId: c.MsgDstId,
		Data:   reData,
	})
}

func (c *ResponseWriter) GetConn() conn.Conn {
	return c.Conn
}

package net

import (
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
)

func NewContext(connect *Conn, message *message.Message) *Context {
	return &Context{
		msg:  message,
		conn: connect,
	}
}

type Context struct {
	msg        *message.Message
	conn       *Conn
	isResponse bool
}

func (c *Context) Type() uint8 {
	return c.msg.Type
}

func (c *Context) Hop() uint8 {
	return c.msg.Hop
}

func (c *Context) Id() uint32 {
	return c.msg.Id
}

func (c *Context) SrcId() uint32 {
	return c.msg.SrcId
}

func (c *Context) DestId() uint32 {
	return c.msg.DestId
}

func (c *Context) Data() []byte {
	return c.msg.Data
}

func (c *Context) String() string {
	return c.msg.String()
}

// Response 响应数据，type为 message.MsgType_Reply，限制回复一次，不要尝试多次回复，多次回复返回 var ErrLimitReply = errors.New("limit reply to one time")
func (c *Context) Response(code int16, data []byte) error {
	if c.isResponse {
		return ErrMultipleResponse
	}
	c.isResponse = true
	c.msg.Hop = 0
	c.msg.Type = message.MsgType_Reply
	c.msg.SrcId, c.msg.DestId = c.conn.localId, c.msg.SrcId
	reData := make([]byte, 2+len(data))
	reData[0] = byte(code)
	reData[1] = byte(code >> 8)
	copy(reData[2:], data)
	c.msg.Data = reData
	return c.conn.SendMessage(c.msg)
}

func (c *Context) Conn() iface.Conn {
	return c.conn
}

// IsResponse 是否已经响应过
func (c *Context) IsResponse() bool {
	return c.isResponse
}

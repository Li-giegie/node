package net

import (
	"encoding/binary"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
)

type Context struct {
	*message.Message
	*Connect
	once bool
	Next bool
}

func (c *Context) Type() uint8 {
	return c.Message.Type
}

func (c *Context) Hop() uint8 {
	return c.Message.Hop
}

func (c *Context) Id() uint32 {
	return c.Message.Id
}

func (c *Context) SrcId() uint32 {
	return c.Message.SrcId
}

func (c *Context) DestId() uint32 {
	return c.Message.DestId
}

func (c *Context) Data() []byte {
	return c.Message.Data
}

func NewContext(connect *Connect, message *message.Message) *Context {
	return &Context{
		Message: message,
		Connect: connect,
		Next:    true,
	}
}

// Reply 响应内容，限制回复一次，不要尝试多次回复，多次回复返回 var ErrLimitReply = errors.New("limit reply to one time")
func (c *Context) Reply(data []byte) (err error) {
	return c.reply(message.MsgType_Reply, data)
}

func (c *Context) ReplyError(err error, data []byte) error {
	var edata []byte
	if err == nil {
		edata = []byte{255, 255, 255, 255}
	} else if err != nil {
		eBuf := []byte(err.Error())
		if 4+len(eBuf)+len(data) >= int(c.maxMsgLen) {
			return ErrMaxMsgLen
		}
		edata = make([]byte, 4+len(eBuf))
		binary.LittleEndian.PutUint32(edata[:4], uint32(len(eBuf)))
		copy(edata[4:], eBuf)
	}
	rdata := make([]byte, len(edata)+len(data))
	copy(rdata, edata)
	copy(rdata[len(edata):], data)
	return c.reply(message.MsgType_ReplyErr, rdata)
}

func (c *Context) Conn() iface.Conn {
	return c.Connect
}

func (c *Context) reply(typ uint8, data []byte) (err error) {
	if c.once {
		return ErrOnce
	}
	c.once = true
	c.Message.Hop = 0
	c.Message.Type = typ
	c.Message.SrcId, c.Message.DestId = c.Message.DestId, c.Message.SrcId
	c.Message.Data = data
	_, err = c.WriteMessage(c.Message)
	return err
}

// Stop 终止进入下一个回调
func (c *Context) Stop() {
	c.Next = false
}

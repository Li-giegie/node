package net

import (
	"encoding/binary"
	"github.com/Li-giegie/node/message"
)

type Context struct {
	*message.Message
	*Connect
	once bool
	next bool
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

func NewContext(connect *Connect, message *message.Message, next bool) *Context {
	return &Context{
		Message: message,
		Connect: connect,
		next:    next,
	}
}

// Reply 响应内容，限制回复一次，不要尝试多次回复，多次回复返回 var ErrLimitReply = errors.New("limit reply to one time")
func (c *Context) Reply(data []byte) (err error) {
	return c.ReplyCustom(message.MsgType_Reply, data)
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
	return c.ReplyCustom(message.MsgType_ReplyErr, rdata)
}

func (c *Context) ReplyCustom(typ uint8, data []byte) (err error) {
	if c.once {
		return ErrOnce
	}
	c.once = true
	c.Message.Hop = 0
	c.Message.Type = typ
	c.Message.SrcId, c.Message.DestId = c.Message.DestId, c.Message.SrcId
	c.Message.Data = data
	_, err = c.WriteMsg(c.Message)
	return err
}

// Next 获取是否进入下一个回调
func (c *Context) Next() bool {
	return c.next
}

// Stop 是否进入下一个回调
func (c *Context) Stop() {
	c.next = false
}

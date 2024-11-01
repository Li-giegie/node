package node

import "github.com/Li-giegie/node/net"

type IMessage interface {
	Id() uint32
	Type() uint8
	SrcId() uint32
	DestId() uint32
	Data() []byte
	String() string
}

type Context interface {
	IMessage
	// Reply 回复内容，每次请求限制回复一次，不要尝试多次回复，多次回复返回 OnceErr = errors.New("write only")
	Reply(data []byte) error
	// ErrReply 回复内容，每次请求限制回复一次，err 的长度限制 (err.Error()) 长度限制 math.MaxUint16-2 (65533)
	ErrReply(data []byte, err error) error
}

type CustomContext interface {
	IMessage
	// CustomReply 回复内容，每次请求限制回复一次，自定义类型回复，适用需要修改消息类型的自定义发送的消息
	CustomReply(typ uint8, data []byte) error
}

type connContext struct {
	*net.Message
	*net.Connect
	once bool
}

func (c *connContext) Id() uint32 {
	return c.Message.Id
}

func (c *connContext) Type() uint8 {
	return c.Message.Type
}

func (c *connContext) SrcId() uint32 {
	return c.Message.SrcId
}

func (c *connContext) DestId() uint32 {
	return c.Message.DestId
}

func (c *connContext) Data() []byte {
	return c.Message.Data
}

// Reply 响应内容，限制回复一次，不要尝试多次回复，多次回复返回 var ErrLimitReply = errors.New("limit reply to one time")
func (c *connContext) Reply(data []byte) (err error) {
	return c.CustomReply(net.MsgType_Reply, data)
}

const maxErrReplySize = 65533

// ErrReply err length <= 65533 byte
func (c *connContext) ErrReply(data []byte, err error) error {
	var errB = make([]byte, 2)
	if err == nil {
		errB[0], errB[1] = 255, 255 //65535
	} else {
		errBytes := []byte(err.Error())
		if len(errBytes) > maxErrReplySize {
			return net.DEFAULT_ErrReplyErrorLengthOverflow
		}
		errB[0], errB[1] = byte(len(errBytes)), byte(len(errBytes)>>8)
		errB = append(errB, errBytes...)
	}
	return c.CustomReply(net.MsgType_ReplyErr, append(errB, data...))
}

func (c *connContext) CustomReply(typ uint8, data []byte) (err error) {
	if c.once {
		return net.DEFAULT_ErrReplyLimitOnce
	}
	c.once = true
	c.Message.Type = typ
	c.Message.SrcId, c.Message.DestId = c.Message.DestId, c.Message.SrcId
	c.Message.Data = data
	_, err = c.WriteMsg(c.Message)
	return err
}

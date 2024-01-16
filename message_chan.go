package node

import utils "github.com/Li-giegie/go-utils"

type iMessageChan interface {
	AddMsgChan(id uint32, msgChan chan *message)
	DeleteMsgChan(id uint32)
	GetMsgChan(id uint32) (chan *message, bool)
}

func newMessageChan() iMessageChan {
	mc := new(messageChan)
	mc.MapUint32 = utils.NewMapUint32()
	return mc
}

type messageChan struct {
	*utils.MapUint32
}

func (m *messageChan) AddMsgChan(id uint32, msgChan chan *message) {
	m.Set(id, msgChan)
}

func (m *messageChan) DeleteMsgChan(id uint32) {
	m.Delete(id)
}

func (m *messageChan) GetMsgChan(id uint32) (chan *message, bool) {
	v, ok := m.Get(id)
	if !ok {
		return nil, false
	}
	return v.(chan *message), true
}

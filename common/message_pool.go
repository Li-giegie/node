package common

import "sync/atomic"

func NewMsgPool(cacheMaxNum int) *MsgPool {
	mm := new(MsgPool)
	mm.p = NewPool(cacheMaxNum, func() any {
		return new(Message)
	})
	return mm
}

type MsgPool struct {
	p          *Pool
	msgIdCount uint32
}

func (mm *MsgPool) RecycleMsg(m *Message) {
	m.Id = 0
	m.Data = nil
	m.Type = 0
	m.SrcId = 0
	m.DestId = 0
	mm.p.Put(m)
}

func (mm *MsgPool) NewMsg(srcId, dstId uint16, typ uint8, data []byte) *Message {
	m := mm.p.Get().(*Message)
	m.Id = atomic.AddUint32(&mm.msgIdCount, 1)
	if m.Id > 0x00AFFFFF {
		mm.msgIdCount = 0
	}
	m.SrcId = srcId
	m.DestId = dstId
	m.Type = typ
	m.Data = data
	return m
}
func (mm *MsgPool) DefaultMsg() *Message {
	return mm.p.Get().(*Message)
}

package common

import "sync/atomic"

func NewMessageConstructor(cacheMaxNum int) *Constructor {
	mm := new(Constructor)
	mm.Pool = NewPool(cacheMaxNum, func() any {
		return new(Message)
	})
	return mm
}

type Constructor struct {
	*Pool
	msgIdCount uint32
}

func (mm *Constructor) Recycle(m *Message) {
	m.Id = 0
	m.Api = 0
	m.Data = nil
	m.Typ = 0
	m.SrcId = 0
	m.DestId = 0
	m.DataLength = 0
	mm.Put(m)
}

func (mm *Constructor) Recycles(m ...*Message) {
	for i := 0; i < len(m); i++ {
		mm.Recycle(m[i])
	}
}

func (mm *Constructor) New(srcId, dstId uint16, typ uint8, api uint16, data []byte) *Message {
	m := mm.Get().(*Message)
	m.Id = atomic.AddUint32(&mm.msgIdCount, 1)
	if m.Id > 0x00AFFFFF {
		mm.msgIdCount = 0
	}
	m.SrcId = srcId
	m.DestId = dstId
	m.Typ = typ
	m.Api = api
	m.Data = data
	return m
}
func (mm *Constructor) Default() *Message {
	return mm.Get().(*Message)
}

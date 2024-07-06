package common

import (
	"sync"
)

type MsgReceiver struct {
	cache map[uint32]chan *Message
	lock  sync.RWMutex
	pool  *Pool
}

func NewMsgReceiver(cap int) *MsgReceiver {
	mr := new(MsgReceiver)
	mr.cache = make(map[uint32]chan *Message)
	mr.pool = NewPool(cap, func() any {
		return make(chan *Message, 1)
	})
	return mr
}

func (m *MsgReceiver) CreateMsgChan(id uint32) chan *Message {
	chanMsg := m.pool.Get().(chan *Message)
	if len(chanMsg) > 0 {
		<-chanMsg
	}
	m.lock.Lock()
	m.cache[id] = chanMsg
	m.lock.Unlock()
	return chanMsg
}

func (m *MsgReceiver) SetMsgChan(msg *Message) bool {
	m.lock.Lock()
	chanM, ok := m.cache[msg.Id]
	if ok {
		chanM <- msg
	}
	m.lock.Unlock()
	return ok
}

func (m *MsgReceiver) DeleteMsgChan(id uint32) bool {
	m.lock.Lock()
	chanM, ok := m.cache[id]
	if !ok {
		m.lock.Unlock()
		return false
	}
	if len(chanM) > 0 {
		<-chanM
	}
	delete(m.cache, id)
	if !m.pool.Put(chanM) {
		close(chanM)
	}
	m.lock.Unlock()
	return true
}

package common

import (
	"sync"
	"sync/atomic"
)

type Pool struct {
	index  *int32
	maxNum int32
	pool   *sync.Pool
}

func NewPool(maxNum int, f func() any) *Pool {
	p := new(Pool)
	p.index = new(int32)
	p.maxNum = int32(maxNum)
	p.pool = &sync.Pool{New: f}
	return p
}

func (p *Pool) Get() any {
	if atomic.LoadInt32(p.index) > 0 {
		atomic.AddInt32(p.index, -1)
	}
	return p.pool.Get()
}

func (p *Pool) Put(x any) bool {
	index := atomic.LoadInt32(p.index)
	if index < p.maxNum {
		atomic.AddInt32(p.index, 1)
		p.pool.Put(x)
		return true
	}
	return false
}


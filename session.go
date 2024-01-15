package node

import (
	"fmt"
	utils "github.com/Li-giegie/go-utils"
	"math/rand"
	"sync/atomic"
	"time"
)

const (
	sessionCache_lifeTime  = time.Second * 10
	sessionCache_checktime = time.Second * 10
)

type sessionContent struct {
	unixTime int64
	tag      string
}

type sessionCache struct {
	count uint32
	cache *utils.MapUint32
	state bool
}

func newSessionCache(lifeTime time.Duration, checkTime time.Duration) *sessionCache {
	s := new(sessionCache)
	s.cache = utils.NewMapUint32()
	s.count = rand.Uint32()
	s.state = true
	go func() {
		for s.state {
			time.Sleep(checkTime)
			key := s.cache.KeyToSlice()
			for _, k := range key {
				v, ok := s.cache.Get(k)
				if ok {
					t, ok := v.(*sessionContent)
					if ok && time.Now().Unix() >= int64(lifeTime.Seconds())+t.unixTime {
						s.cache.Delete(k)
						fmt.Println("删除", k, t.unixTime, t.tag)
					}
				}
			}

		}
	}()
	return s
}

func (s *sessionCache) create(addr string) uint32 {
	n := atomic.AddUint32(&s.count, 1)
	s.cache.Set(n, &sessionContent{
		unixTime: time.Now().Unix(),
		tag:      addr,
	})
	return n
}

func (s *sessionCache) delete(n uint32) {
	s.cache.Delete(n)
}

func (s *sessionCache) query(n uint32) (*sessionContent, bool) {
	v, ok := s.cache.Get(n)
	if !ok {
		return nil, false
	}
	se, ok := v.(*sessionContent)
	return se, ok
}

func (s *sessionCache) stopTimeoutCheck() {
	s.state = false
}

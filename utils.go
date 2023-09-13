package node

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"
)

var _rnd *rand.Rand

func init() {
	_rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// 1024-49151
func getPort() string {
	return strconv.Itoa(_rnd.Intn(49152-1024) + 1024)
}

// 测试用：原子计数器
type Counter struct {
	requestNum uint64
	replyNum   uint64
	t          time.Time
}

func (c *Counter) AddRequestNum() uint64 {
	return atomic.AddUint64(&c.requestNum, 1)
}

func (c *Counter) AddReplyNum() uint64 {
	return atomic.AddUint64(&c.replyNum, 1)
}

func NewCounter() *Counter {
	var c = new(Counter)
	c.t = time.Now()
	return c
}

func (c *Counter) String() string {
	return fmt.Sprintf("request num:[%v],reply num:[%v]", c.requestNum, c.replyNum)
}

func (c *Counter) Debug() {
	fmt.Printf("耗时 %v 效率 %v\n", time.Since(c.t), c.String())
}

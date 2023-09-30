package node

import (
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"math/rand"
	"net"
	"os"
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
	sendNum    uint64
	t          time.Time
}

func (c *Counter) AddRequest() uint64 {
	return atomic.AddUint64(&c.requestNum, 1)
}

func (c *Counter) AddSend() uint64 {
	return atomic.AddUint64(&c.sendNum, 1)
}

func NewCounter() *Counter {
	var c = new(Counter)
	c.t = time.Now()
	return c
}

func (c *Counter) String() string {
	return fmt.Sprintf("request num:[%v],send num:[%v]", c.requestNum, c.sendNum)
}

func (c *Counter) Debug() {
	fmt.Printf("耗时 %v 效率 %v\n", time.Since(c.t), c.String())
}

func readMessage(conn *net.TCPConn) (*MessageBase, error) {
	buf, err := jeans.Unpack(conn)
	if err != nil {
		return nil, err
	}
	msg := NewMessageBaseWithUnmarshal(buf)
	return msg, nil
}

func write(conn *net.TCPConn, buf []byte) error {
	_, err := conn.Write(jeans.Pack(buf))
	return err
}

func mustAddress(protocol string, addrs ...string) []*net.TCPAddr {
	var addr = make([]*net.TCPAddr, 0, len(addrs))
	for _, item := range addrs {
		tmp, err := net.ResolveTCPAddr(protocol, item)
		if err != nil {
			fmt.Printf("address or protocol format error %v", err)
			os.Exit(0)
		}
		addr = append(addr, tmp)
	}
	return addr
}

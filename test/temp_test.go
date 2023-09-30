package test

import (
	"bytes"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"github.com/Li-giegie/node"
	"runtime"
	"sync"
	"testing"
)

func TestPack(t *testing.T) {
	buf := jeans.Pack(nil)
	fmt.Println(buf, len(buf))
	r := bytes.NewBuffer(buf)
	buf, err := jeans.Unpack(r)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(buf, len(buf))

}

func TestSyncMap(t *testing.T) {
	var m sync.Map
	m.Store(1, 1)
	m.Store("p", 2)
	fmt.Println(m.Load(1))
	fmt.Println(m.Load("p"))
}

func TestGetCPUCore(t *testing.T) {
	fmt.Println(runtime.NumCPU())
}

func TestWorkerProcess(t *testing.T) {
	var m = map[int]int{}

	fmt.Println(m[1])

	var b byte = '0'
	var c byte = 0
	fmt.Println(b, c, b == c)
}

func TestMsgForwardMarshal(t *testing.T) {
	m1 := new(node.MessageForward)
	m1.Data = []byte("123-a")
	m1.DestId = "dest-id"
	m1.SrcId = "src-id"
	m2 := new(node.MessageForward)
	m2.Unmarshal(m1.Marshal())
	fmt.Println(m2)
}

func TestMsgForwardMarshalScene(t *testing.T) {
	m1 := node.NewMessageForward("1", "2", []byte("forward"))
	fmt.Println(m1.Marshal())
	m2 := new(node.MessageForward)
	m2.Unmarshal(m1.Marshal())
	fmt.Println(m2)
}

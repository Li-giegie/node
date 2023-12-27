package handle

import (
	"fmt"
	"github.com/Li-giegie/node"
	"testing"
)

func TestNewHandler(t *testing.T) {
	h := node.NewHandler()
	h.Add(1, node.HandlerFuncSend(func(id uint64, data []byte) {
		fmt.Println("send handle ", id, string(data))
	}))
	h.Add(2, node.HandlerFuncRequest(func(id uint64, data []byte) ([]byte, error) {
		fmt.Println("request handle ", id, string(data))
		return nil, nil
	}))
	ih, _ := h.Get(1)
	switch ih.Typ() {
	case node.HandlerTYpe_Send:
		ih.(node.HandlerFuncSend)(1, nil)
	case node.HandlerTYpe_Request:
		reply, err := ih.(node.HandlerFuncRequest)(2, nil)
		fmt.Println(string(reply), err)
	default:
		panic("example")
	}
}

func BenchmarkNewHandler(b *testing.B) {
	h := node.NewHandler()
	h.Add(1, node.HandlerFuncSend(func(id uint64, data []byte) {

	}))
	h.Add(2, node.HandlerFuncRequest(func(id uint64, data []byte) ([]byte, error) {

		return nil, nil
	}))
	for i := 0; i < b.N; i++ {
		h.Get(uint32(i))
	}
}

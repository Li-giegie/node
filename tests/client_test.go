package tests

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/client"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/reply"
	"log"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	c := node.NewClientOption(2, 1,
		client.WithRemoteKey([]byte("hello")),
	)
	client.OnMessage(func(r *reply.Reply, m *message.Message) (next bool) {
		log.Println("OnMessage", m.String())
		return true
	})
	err := c.Connect("tcp://127.0.0.1:8000", nil)
	if err != nil {
		log.Fatalln(err)
	}
	t1 := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < 1000000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			code, resp, err := c.Request(context.Background(), []byte("ping"+strconv.Itoa(i)))
			if err != nil {
				log.Println(err, code, resp)
				return
			}
		}(i)
	}
	wg.Wait()
	fmt.Println(time.Since(t1))
}

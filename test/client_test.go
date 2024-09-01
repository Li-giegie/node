package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"log"
	"strconv"
	"sync"
	"testing"
	"time"
)

type CliHandler struct {
}

func (h CliHandler) Connection(conn common.Conn) {
	log.Println("Handle", conn.RemoteId())
}

func (h CliHandler) Handle(ctx common.Context) {
	log.Println("Handle", ctx.String())
	if err := ctx.Reply(ctx.Data()); err != nil {
		fmt.Println(err)
	}
}

func (h CliHandler) ErrHandle(ctx common.ErrContext, err error) {
	log.Println("ErrHandle", err, ctx.String())
}

func (h CliHandler) CustomHandle(ctx common.CustomContext) {
	log.Println("CustomHandle", ctx.String())
	ctx.CustomReply(ctx.Type(), ctx.Data())

}

func (h CliHandler) Disconnect(id uint16, err error) {
	log.Println("Disconnect", id, err)
}

func TestClient(t *testing.T) {
	conn, err := node.DialTCP(context.Background(), 8001, 8000, "0.0.0.0:8000", &CliHandler{})
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()

	t1 := time.Now()
	wg := sync.WaitGroup{}
	for i := 0; i < 100000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			data := []byte(strconv.Itoa(i))
			res, err := conn.Request(context.Background(), data)
			if err != nil {
				t.Error(err, res)
				return
			}
			if string(res) != string(data) {
				log.Fatalln("值被修改", string(res), string(data), res, data)
			}
		}(i)
	}
	wg.Wait()
	fmt.Println(time.Since(t1))
}

func TestAAA(t *testing.T) {
	c := make(chan int, 1)

	go func() {
		time.Sleep(time.Second)
		close(c)
	}()
	select {
	case v := <-c:
		fmt.Println(v)
	}
}

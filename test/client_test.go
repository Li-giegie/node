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
	conn, err := node.DialTCP(
		"0.0.0.0:8000",
		&node.Identity{
			Id:            8001,
			AccessKey:     []byte("hello"),
			AccessTimeout: time.Second * 6,
		},
		&CliHandler{},
	)
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()

	t1 := time.Now()
	wg := sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			data := []byte(strconv.Itoa(i))
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			res, err := conn.Request(ctx, data)
			if err != nil {
				log.Println("err", err, res)
			} else {
				if string(res) != string(data) {
					log.Println("值被修改", string(res), string(data), res, data)
				}
			}
			wg.Done()
			cancel()
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

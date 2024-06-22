package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"log"
	"net"
	"sync"
	"testing"
	"time"
)

type ClientHandle struct {
	common.Conn
	stopC chan error
}

func (c ClientHandle) Connection(conn net.Conn) (id uint16, err error) {
	log.Println("Connection ", conn.RemoteAddr().String())
	return 0, err
}

func (c ClientHandle) Handle(ctx *common.Context) {
	go func() {
		log.Println("Handle ", ctx.String())
		if err := ctx.Reply([]byte("client 1 handle success")); err != nil {
			fmt.Println(err)
		}
	}()
}

func (c ClientHandle) ErrHandle(msg *common.Message) {
	log.Println("ErrHandle ", msg.String())
}

func (c ClientHandle) DropHandle(msg *common.Message) {
	log.Println("DropHandle ", msg.String())
}

func (c ClientHandle) CustomHandle(ctx *common.Context) {
	log.Println("CustomHandle ", ctx.String())
}

func (c ClientHandle) Disconnect(id uint16, err error) {
	log.Println("Disconnect ", id, err)
	c.stopC <- err
}

func TestClient(t *testing.T) {
	c := new(ClientHandle)
	c.stopC = make(chan error, 1)
	conn, err := node.Dial("tcp", "0.0.0.0:8080", 1, c)
	if err != nil {
		t.Error(err)
		return
	}
	c.Conn = conn
	fmt.Println("开始发送")
	wg := sync.WaitGroup{}
	t1 := time.Now()
	conn.WriteMsg(&common.Message{
		Type:   20,
		Id:     0,
		SrcId:  0,
		DestId: 0,
		Data:   nil,
	})
	conn.Close()
	return
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := conn.Request(context.Background(), []byte("client hello 1"))
			if err != nil {
				t.Error(err, data)
				return
			}
			fmt.Println("收到", string(data))
		}()
	}
	wg.Wait()
	fmt.Println(time.Since(t1))
	c.Close()
	fmt.Println("结束")
	<-c.stopC
}

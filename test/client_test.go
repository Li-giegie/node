package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/common"
	"github.com/Li-giegie/node/pkg/conn"
	context2 "github.com/Li-giegie/node/pkg/ctx"
	"github.com/Li-giegie/node/pkg/message"
	"log"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	stopC := make(chan struct{}, 1)
	c := node.NewClient(8001, &common.Identity{Id: 8000, Key: []byte("hello"), AuthTimeout: time.Second * 6})
	c.OnConnect(func(conn conn.Conn) {

	})
	c.OnMessage(func(ctx context2.Context) {

	})
	c.OnClose(func(conn conn.Conn, err error) {
		stopC <- struct{}{}
	})
	c.SetKeepalive(time.Second*1, time.Second*3, time.Second*5)
	err := c.Connect("tcp://127.0.0.1:8000")
	if err != nil {
		log.Fatalln(err)
	}
	res, code, err := c.Request(context.Background(), []byte("ping"))
	fmt.Printf("1 Request res %s ,err %v code %d\n", res, err, code)
	res, code, err = c.RequestTo(context.Background(), 5, []byte("hello"))
	fmt.Printf("2 Request res %s ,err %v code %d\n", res, err, code)
	res, code, err = c.RequestType(context.Background(), message.MsgType_Undefined, []byte("hello"))
	fmt.Printf("3 Request res %s ,err %v code %d\n", res, err, code)
	res, code, err = c.RequestType(context.Background(), message.MsgType_Undefined, make([]byte, 120))
	fmt.Printf("4 Request res %s ,err %v code %d\n", res, err, code)
	_ = c.Close()
	fmt.Println("close")
	<-stopC

	for {

	}
}

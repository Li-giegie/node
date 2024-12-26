package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	"log"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	stopC := make(chan struct{}, 1)
	c := node.NewClient(8001, &node.Identity{Id: 8000, Key: []byte("hello"), AuthTimeout: time.Second * 6})
	c.OnConnect(func(conn iface.Conn) {

	})
	c.OnMessage(func(ctx iface.Context) {

	})
	c.OnClose(func(conn iface.Conn, err error) {
		stopC <- struct{}{}
	})
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
}

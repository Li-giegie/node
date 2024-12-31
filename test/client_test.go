package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/client"
	"github.com/Li-giegie/node/pkg/conn"
	context2 "github.com/Li-giegie/node/pkg/ctx"
	"github.com/Li-giegie/node/pkg/message"
	"log"
	"testing"
)

func TestClient(t *testing.T) {
	stopC := make(chan struct{}, 1)
	c := node.NewClientOption(1, 10, client.WithRemoteKey([]byte("hello")))
	c.OnConnect(func(conn conn.Conn) {

	})
	c.OnMessage(func(ctx context2.Context) {

	})
	c.OnClose(func(conn conn.Conn, err error) {
		stopC <- struct{}{}
	})
	err := c.Connect("tcp://127.0.0.1:8000")
	if err != nil {
		log.Fatalln(err)
	}
	resp, code, err := c.Request(context.Background(), []byte("ping"))
	fmt.Printf("1 Request res %v,code %d ,err %v \n", resp, code, err)
	resp, code, err = c.RequestTo(context.Background(), 5, []byte("hello"))
	fmt.Printf("2 Request res %v,code %d ,err %v \n", resp, code, err)
	resp, code, err = c.RequestType(context.Background(), message.MsgType_Undefined, []byte("hello"))
	fmt.Printf("3 Request res %v,code %d ,err %v \n", resp, code, err)
	resp, code, err = c.RequestType(context.Background(), message.MsgType_Undefined, make([]byte, 120))
	fmt.Printf("4 Request res %v,code %d ,err %v \n", resp, code, err)
	_ = c.Close()
	fmt.Println("close")
	<-stopC
}

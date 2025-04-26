package tests

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/client"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"log"
	"strings"
	"testing"
)

func TestClient(t *testing.T) {
	c := node.NewClientOption(2, 1,
		client.WithRemoteKey([]byte("hello")),
	)
	c.OnMessage(func(r responsewriter.ResponseWriter, m *message.Message) {

		fmt.Println("OnMessage", m.String())
	})
	c.OnClose(func(err error) {
		fmt.Println("OnClose", err)
	})
	err := c.Connect("tcp://127.0.0.1:8000")
	if err != nil {
		log.Fatalln(err)
	}
	resp, code, err := c.Request(context.Background(), []byte("ping"))
	fmt.Printf("1 Request res %s,code %d ,err %v \n", resp, code, err)
	resp, code, err = c.RequestTo(context.Background(), 5, []byte("hello"))
	fmt.Printf("2 Request res %s,code %d ,err %v \n", resp, code, err)
	resp, code, err = c.RequestType(context.Background(), message.MsgType_Undefined, []byte("hello"))
	fmt.Printf("3 Request res %s,code %d ,err %v \n", resp, code, err)
	resp, code, err = c.RequestType(context.Background(), message.MsgType_Undefined, make([]byte, 5))
	fmt.Printf("4 Request res %s,code %d ,err %v \n", resp, code, err)
	_ = c.Close()
	fmt.Println("close")
}

func TestName(t *testing.T) {
	fmt.Println(len(strings.Fields("")))
}

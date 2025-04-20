package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/client"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"log"
	"net"
	"testing"
)

func TestClient(t *testing.T) {
	stopC := make(chan struct{}, 1)
	c := node.NewClientOption(2, 2,
		client.WithRemoteKey([]byte("hello")),
	)
	c.OnAccept(func(conn net.Conn) (next bool) {
		fmt.Println("OnAccept", conn.RemoteAddr())
		return true
	})
	c.OnConnect(func(conn conn.Conn) bool {
		fmt.Println("OnConnect", conn.RemoteId())
		return true
	})
	c.OnMessage(func(w responsewriter.ResponseWriter, m *message.Message) (next bool) {
		w.Response(message.StateCode_Success, m.Data)
		return true
	})
	c.OnClose(func(conn conn.Conn, err error) bool {
		stopC <- struct{}{}
		return true
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
	<-stopC
}

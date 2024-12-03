package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	"net"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	netConn, err := net.Dial("tcp", "0.0.0.0:8000")
	if err != nil {
		t.Error(err)
		return
	}
	stopC := make(chan struct{})
	c := node.NewClient(8001, &node.Identity{Id: 8000, Key: []byte("hello"), AuthTimeout: time.Second * 6}, nil)
	c.AddOnMessage(func(ctx iface.Context) {
		fmt.Println(ctx.String())
		ctx.Reply(ctx.Data())
	})
	c.AddOnClose(func(conn iface.Conn, err error) {
		fmt.Println("OnClose", err)
		stopC <- struct{}{}
	})
	conn, err := c.Start(netConn)
	if err != nil {
		t.Error(err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	res, err := conn.Request(ctx, []byte("ping"))
	fmt.Printf("Request res %s ,err %v\n", res, err)
	res, err = conn.RequestTo(context.Background(), 5, []byte("hello"))
	fmt.Printf("RequestTo res %s ,err %v\n", res, err)
	res, err = conn.RequestType(context.Background(), message.MsgType_Undefined, []byte("hello"))
	fmt.Printf("RequestType res %s ,err %v\n", res, err)
	_ = conn.Close()
	<-stopC
}

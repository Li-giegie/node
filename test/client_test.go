package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
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
	c := node.NewClient(8001, &node.Identity{Id: 8000, Key: []byte("hello"), Timeout: time.Second * 6}, nil)
	c.AddOnMessage(func(ctx iface.Context) {
		fmt.Println(ctx.String())
		ctx.Reply(ctx.Data())
	})
	c.AddOnClose(func(conn iface.Conn, err error) {
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
	fmt.Println(string(res), err)
	_ = conn.Close()
	<-stopC
}

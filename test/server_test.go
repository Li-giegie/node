package test

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"log"
	"net"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	srv := node.NewServer(&node.Identity{Id: 8000, Key: []byte("hello"), Timeout: time.Second * 6}, nil)
	srv.AddOnConnect(func(conn iface.Conn) {
		log.Println("OnConnection", conn.RemoteId())
	})
	srv.AddOnMessage(func(ctx iface.Context) {
		log.Println("OnMessage", ctx.String())
		fmt.Println(ctx.Reply(ctx.Data()))
	})
	l, err := net.Listen("tcp", "0.0.0.0:8000")
	if err != nil {
		t.Error(err)
		return
	}
	if err = srv.Serve(l); err != nil {
		t.Error(err)
	}
}

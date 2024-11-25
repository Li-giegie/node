package test

import (
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"log"
	"net"
	"testing"
	"time"
)

func TestEchoServer(t *testing.T) {
	srv := node.NewServer(&node.Identity{Id: 8000, Key: []byte("hello"), Timeout: time.Second * 6}, nil)
	srv.AddOnConnection(func(conn iface.Conn) {
		log.Println("OnConnection", conn.RemoteId(), conn.NodeType())
	})
	srv.AddOnMessage(func(ctx iface.Context) {
		ctx.Reply(ctx.Data())
	})
	srv.AddOnCustomMessage(func(ctx iface.Context) {
		log.Println("OnCustomMessage", ctx.String())
	})
	srv.AddOnClosed(func(conn iface.Conn, err error) {
		log.Println(conn.RemoteId(), err, conn.NodeType())
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

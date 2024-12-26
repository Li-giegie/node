package test

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	"net"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	srv := node.NewServer(&node.Identity{Id: 8000, Key: []byte("hello"), AuthTimeout: time.Second * 6})
	handle := node.NewHandler(srv)
	handle.OnAccept(func(conn net.Conn) (allow bool) {
		fmt.Println("OnAccept")
		return true
	})
	handle.OnConnect(func(conn iface.Conn) {
		fmt.Println("OnConnection")
	})
	handle.OnMessage(func(ctx iface.Context) {
		fmt.Println("OnMessage")
		ctx.Response(message.StateCode_Success, ctx.Data())
	})
	handle.OnClose(func(conn iface.Conn, err error) {
		fmt.Println("OnClose", conn.RemoteId())
	})
	handle.Register(1, &node.EmptyHandler{})
	if err := srv.ListenAndServe("0.0.0.0:8000"); err != nil {
		t.Error(err)
	}
}

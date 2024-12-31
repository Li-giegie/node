package test

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/ctx"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/server"
	"net"
	"testing"
)

func TestServer(t *testing.T) {
	srv := node.NewServerOption(10,
		server.WithAuthKey([]byte("hello")),
	)
	srv.OnAccept(func(conn net.Conn) (allow bool) {
		fmt.Println("OnAccept")
		return true
	})
	srv.OnConnect(func(conn conn.Conn) {
		fmt.Println("OnConnection")
	})
	srv.OnMessage(func(ctx ctx.Context) {
		fmt.Println("OnMessage")
		ctx.Response(message.StateCode_Success, ctx.Data())
	})
	srv.OnClose(func(conn conn.Conn, err error) {
		fmt.Println("OnClose", conn.RemoteId())
	})

	fmt.Println("listening on :8000")
	if err := srv.ListenAndServe("0.0.0.0:8000"); err != nil {
		t.Error(err)
	}
}

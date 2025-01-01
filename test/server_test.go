package test

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/handler"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"github.com/Li-giegie/node/pkg/server"
	"net"
	"testing"
)

func TestServer(t *testing.T) {
	srv := node.NewServerOption(1,
		server.WithAuthKey([]byte("hello")),
	)
	srv.OnAccept(func(conn net.Conn) (next bool) {
		fmt.Println("OnAccept")
		return true
	})
	srv.OnConnect(func(conn conn.Conn) (next bool) {
		fmt.Println("OnConnection")
		return true
	})
	srv.OnMessage(func(r responsewriter.ResponseWriter, m *message.Message) (next bool) {
		r.Response(message.StateCode_Success, m.Data)
		return false
	})
	srv.Register(message.MsgType_Default, &handler.Default{
		OnAcceptFunc:  nil,
		OnConnectFunc: nil,
		OnMessageFunc: func(r responsewriter.ResponseWriter, m *message.Message) {
			fmt.Println(r.Response(message.StateCode_Success, m.Data))
		},
		OnCloseFunc: nil,
	})
	srv.OnClose(func(conn conn.Conn, err error) (next bool) {
		fmt.Println("OnClose", err)
		return true
	})
	fmt.Println("listening on :8000")
	if err := srv.ListenAndServe("0.0.0.0:8000"); err != nil {
		t.Error(err)
	}
}

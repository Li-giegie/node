package tests

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/reply"
	"github.com/Li-giegie/node/pkg/server"
	"net"
	"testing"
)

func TestServer(t *testing.T) {
	srv := node.NewServerOption(1,
		server.WithAuthKey([]byte("hello")),
	)
	server.OnAccept(func(conn net.Conn) (next bool) {
		fmt.Println("OnAccept")
		return true
	})
	server.OnConnect(func(conn *conn.Conn) (next bool) {
		fmt.Println("OnConnection")
		return true
	})
	server.OnMessage(func(r *reply.Reply, m *message.Message) (next bool) {
		//fmt.Println("OnMessage", m.String())
		r.Write(message.StateCode_Success, m.Data)
		return false
	})
	server.OnClose(func(conn *conn.Conn, err error) (next bool) {
		fmt.Println("OnClose", err)
		return true
	})
	fmt.Println("listening on :8000")
	if err := srv.ListenAndServe("0.0.0.0:8000", nil); err != nil {
		t.Error(err)
	}
}

package test

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/common"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/ctx"
	"github.com/Li-giegie/node/pkg/message"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	a := uint32(1)
	fmt.Println(a)
	fmt.Println(atomic.AddUint32(&a, 1))
	fmt.Println(a)
	return
	srv := node.NewServer(&common.Identity{Id: 8000, Key: []byte("hello"), AuthTimeout: time.Second * 6})
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
	srv.SetKeepalive(time.Second*5, time.Second*10, time.Second*20)
	fmt.Println("listening on :8000")
	if err := srv.ListenAndServe("0.0.0.0:8000"); err != nil {
		t.Error(err)
	}
}

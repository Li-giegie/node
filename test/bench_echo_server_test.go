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
	l, err := net.Listen("tcp", "0.0.0.0:8000")
	if err != nil {
		t.Error(err)
		return
	}
	srv := node.NewServer(l, &node.SrvConf{
		Identity: &node.Identity{
			Id:          1,
			AuthKey:     []byte("hello"),
			AuthTimeout: time.Second * 6,
		},
		MaxConns:           0,
		MaxMsgLen:          0xffffff,
		WriterQueueSize:    1024,
		ReaderBufSize:      4096,
		WriterBufSize:      4096,
		MaxListenSleepTime: time.Minute,
		ListenStepTime:     time.Second,
	})
	srv.AddOnConnection(func(conn iface.Conn) {
		log.Println("OnConnection", conn.RemoteId(), conn.NodeType())
	})
	srv.AddOnMessage(func(ctx iface.Context) {
		log.Println("OnMessage", ctx.String())
		ctx.Reply(ctx.Data())
	})
	srv.AddOnCustomMessage(func(ctx iface.Context) {
		log.Println("OnCustomMessage", ctx.String())
	})
	srv.AddOnClosed(func(conn iface.Conn, err error) {
		log.Println(conn.RemoteId(), err, conn.NodeType())
	})
	if err != nil {
		t.Error(err)
		return
	}
	defer srv.Close()
	if err = srv.Serve(); err != nil {
		log.Println(err)
	}
}

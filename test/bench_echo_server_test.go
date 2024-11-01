package test

import (
	"github.com/Li-giegie/node"
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
	srv.OnConnection = func(conn node.Conn) {
		log.Println("OnConnection", conn.RemoteId())
	}
	srv.OnMessage = func(ctx node.Context) {
		//log.Println("OnMessage", ctx.String())
		ctx.Reply(ctx.Data())
	}
	srv.OnCustomMessage = func(ctx node.CustomContext) {
		log.Println("OnCustomMessage", ctx.String())
	}
	srv.OnClose = func(id uint32, err error) {
		log.Println("OnClose", id, err)
	}
	if err != nil {
		t.Error(err)
		return
	}
	defer srv.Close()
	if err = srv.Serve(); err != nil {
		log.Println(err)
	}
}

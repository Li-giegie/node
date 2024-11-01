package test

import (
	"context"
	"github.com/Li-giegie/node"
	"log"
	"net"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	conn, err := net.Dial("tcp", "0.0.0.0:8000")
	if err != nil {
		t.Error(err)
		return
	}
	stopC := make(chan struct{})
	c := node.NewClient(conn, &node.CliConf{
		ReaderBufSize:   4096,
		WriterBufSize:   4096,
		WriterQueueSize: 1024,
		MaxMsgLen:       0xffffff,
		ClientIdentity: &node.ClientIdentity{
			Id:            1234,
			RemoteAuthKey: []byte("hello"),
			Timeout:       time.Second * 6,
		},
	})
	c.OnConnection = func(conn node.Conn) {
		log.Println("OnConnection", conn.RemoteId())
	}
	c.OnMessage = func(ctx node.Context) {
		log.Println("OnMessage", ctx.String())
	}
	c.OnClose = func(id uint32, err error) {
		log.Println("OnClose", id, err)
		stopC <- struct{}{}
	}
	if err = c.Start(); err != nil {
		log.Fatalln(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	res, err := c.Request(ctx, []byte("ping"))
	if err != nil {
		t.Error(err)
		return
	}
	println(string(res))
	c.Close()
	<-stopC
}

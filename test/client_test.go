package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"log"
	"net"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	conn, err := net.Dial("tcp", "0.0.0.0:8001")
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
			Id:            8000,
			RemoteAuthKey: []byte("hello"),
			Timeout:       time.Second * 6,
		},
	})
	c.AddOnMessage(func(conn iface.Context) {
		log.Println(conn.String())
	})
	if err = c.Start(); err != nil {
		log.Fatalln(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	res, err := c.Forward(ctx, 10, []byte("ping"))
	if err != nil {
		fmt.Println(err)
		return
	}
	c.Close()
	println(string(res))
	<-stopC
}

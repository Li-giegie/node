package main

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node/pkg/client/impl_client"
	"github.com/Li-giegie/node/pkg/common"
	"github.com/Li-giegie/node/pkg/conn"
	context2 "github.com/Li-giegie/node/pkg/ctx"
	"log"
	"time"
)

func main() {
	c := impl_client.NewClient(8001, &common.Identity{Id: 8000, Key: []byte("hello"), AuthTimeout: time.Second * 6})
	exitChan := make(chan struct{}, 1)
	c.OnConnect(func(conn conn.Conn) {
		log.Println("OnConnection id", conn.RemoteId(), "type")
	})
	c.OnMessage(func(ctx context2.Context) {
		log.Println("OnMessage", string(ctx.Data()))
		ctx.Response(200, []byte(fmt.Sprintf("from %d data %s", c.Id(), ctx.Data())))
	})
	c.OnClose(func(conn conn.Conn, err error) {
		log.Println("OnClosed", err)
		exitChan <- struct{}{}
	})
	err := c.Connect("0.0.0.0:8000")
	if err != nil {
		log.Fatalln(err)
	}
	res, code, err := c.Request(context.Background(), []byte("hello"))
	fmt.Println(code, string(res), err)
	_ = c.Close()
	<-exitChan
}

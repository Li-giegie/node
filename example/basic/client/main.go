package main

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"log"
	"time"
)

func main() {
	c := node.NewClient(8001, &node.Identity{Id: 8000, Key: []byte("hello"), AuthTimeout: time.Second * 6})
	exitChan := make(chan struct{}, 1)
	c.OnConnect(func(conn iface.Conn) {
		log.Println("OnConnection id", conn.RemoteId(), "type")
	})
	c.OnMessage(func(ctx iface.Context) {
		log.Println("OnMessage", string(ctx.Data()))
		ctx.Response(200, []byte(fmt.Sprintf("from %d data %s", c.Id(), ctx.Data())))
	})
	c.OnClose(func(conn iface.Conn, err error) {
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

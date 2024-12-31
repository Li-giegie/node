package main

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/conn"
	context2 "github.com/Li-giegie/node/pkg/ctx"
	"log"
)

func main() {
	// 创建一个节点为8081的节点
	c := node.NewClientOption(8081, 8000)
	exitChan := make(chan struct{}, 1)
	c.OnConnect(func(conn conn.Conn) {
		log.Println("OnConnection id", conn.RemoteId(), "type")
	})
	c.OnMessage(func(ctx context2.Context) {
		log.Println("OnMessage", string(ctx.Data()))
		ctx.Response(200, []byte(fmt.Sprintf("from %d data %s", c.NodeId(), ctx.Data())))
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

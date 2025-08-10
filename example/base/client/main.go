package main

import (
	"context"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/client"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/reply"
	"log"
)

func main() {
	c := node.NewClientOption(2, 1)
	client.OnMessage(func(r *reply.Reply, m *message.Message) (next bool) {
		log.Println(m.String())
		r.Write(message.StateCode_Success, m.Data)
		return true
	})
	client.OnClose(func(conn *conn.Conn, err error) (next bool) {
		log.Println("OnClose", err)
		return true
	})
	err := c.Connect("tcp://127.0.0.1:7890", nil)
	if err != nil {
		log.Fatal("connect err:", err)
		return
	}
	log.Println("connect: 7890")
	defer c.Close()
	if err = c.Send([]byte("hello")); err != nil {
		log.Println("send err:", err)
		return
	}
	code, data, err := c.Request(context.TODO(), []byte("world"))
	if err != nil {
		log.Println("request err:", err)
		return
	}
	log.Println("code:", code, "data:", string(data))
}

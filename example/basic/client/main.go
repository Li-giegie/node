package main

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"log"
	"net"
)

func main() {
	// 创建一个节点为8081的节点
	c := node.NewClientOption(8081, 8000)
	exitChan := make(chan struct{}, 1)
	c.OnAccept(func(conn net.Conn) (next bool) {
		return true
	})
	c.OnConnect(func(conn conn.Conn) (next bool) {
		return true
	})
	c.OnMessage(func(r responsewriter.ResponseWriter, m *message.Message) (next bool) {
		if m.Type != message.MsgType_Default {
			r.Response(message.StateCode_MessageTypeInvalid, nil)
			return false
		}
		r.Response(message.StateCode_Success, []byte(fmt.Sprintf("response from %d: ok", c.NodeId())))
		return true
	})
	c.OnClose(func(conn conn.Conn, err error) (next bool) {
		return true
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

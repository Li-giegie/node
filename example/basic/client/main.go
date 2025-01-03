package main

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/handler"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"log"
	"net"
)

func main() {
	// 创建一个节点为8081的节点
	c := node.NewClientOption(8081, 8000)
	// OnAccept 注册全局OnAccept回调函数，net.Listen.Accept之后第一个回调函数，同步调用
	c.OnAccept(func(conn net.Conn) (next bool) {
		log.Println("OnAccept", conn.RemoteAddr().String())
		return true
	})
	// OnConnect 注册全局OnConnect回调函数，OnAccept之后的回调函数，同步调用
	c.OnConnect(func(conn conn.Conn) (next bool) {
		log.Println("OnConnect", conn.RemoteAddr().String())
		return true
	})
	// OnMessage 注册全局OnMessage回调函数，OnConnect之后每次收到请求时的回调函数，同步调用
	c.OnMessage(func(r responsewriter.ResponseWriter, m *message.Message) (next bool) {
		log.Println("OnMessage", m.String())
		return true
	})
	// OnClose 注册OnClose回调函数，连接被关闭后的回调函数
	exitChan := make(chan struct{}, 1)
	c.OnClose(func(conn conn.Conn, err error) (next bool) {
		log.Println("OnClose", conn.RemoteAddr().String())
		exitChan <- struct{}{}
		return true
	})
	// Register 注册实现了handler.Handler的处理接口，该接口的回调函数在OnAccept、OnConnect、OnMessage、OnClose之后被回调
	c.Register(message.MsgType_Default, &handler.Default{
		OnAcceptFunc:  nil,
		OnConnectFunc: nil,
		OnMessageFunc: func(r responsewriter.ResponseWriter, m *message.Message) {
			log.Println("OnMessageFunc handle")
			r.Response(message.StateCode_Success, []byte(fmt.Sprintf("response from %d: ok", c.NodeId())))
		},
		OnCloseFunc: nil,
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

package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/handler"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"log"
	"net"
	"os"
)

func main() {
	// 创建8000服务端节点
	s := node.NewServerOption(8000)
	// OnAccept 注册全局OnAccept回调函数，net.Listen.Accept之后第一个回调函数，同步调用
	s.OnAccept(func(conn net.Conn) (next bool) {
		log.Println("OnAccept", conn.RemoteAddr().String())
		return true
	})
	// OnConnect 注册全局OnConnect回调函数，OnAccept之后的回调函数，同步调用
	s.OnConnect(func(conn conn.Conn) (next bool) {
		log.Println("OnConnect", conn.RemoteAddr().String())
		return true
	})
	// OnMessage 注册全局OnMessage回调函数，OnConnect之后每次收到请求时的回调函数，同步调用
	s.OnMessage(func(r responsewriter.ResponseWriter, m *message.Message) (next bool) {
		log.Println("OnMessage", m.String())
		return true
	})
	// OnClose 注册OnClose回调函数，连接被关闭后的回调函数
	s.OnClose(func(conn conn.Conn, err error) (next bool) {
		log.Println("OnClose", conn.RemoteAddr().String())
		return true
	})
	// Register 注册实现了handler.Handler的处理接口，该接口的回调函数在全局OnAccept、OnConnect、OnMessage、OnClose之后被回调
	s.Register(message.MsgType_Default, &handler.Default{
		OnMessageFunc: func(r responsewriter.ResponseWriter, m *message.Message) {
			log.Println("Register OnMessage", m.String())
			r.Response(message.StateCode_Success, append([]byte(fmt.Sprintf("response from %d: ", s.NodeId())), m.Data...))
		},
	})
	go func() {
		scan := bufio.NewScanner(os.Stdin)
		print(">>")
		for scan.Scan() {
			switch scan.Text() {
			case "":
			case "exit":
				print("bye~")
				return
			default:
				for _, c := range s.GetAllConn() {
					res, code, err := c.Request(context.Background(), scan.Bytes())
					if err != nil {
						fmt.Println(err)
					} else {
						fmt.Println(code, string(res))
					}
				}
			}
			print(">>")
		}
	}()
	// 侦听并开启服务
	err := s.ListenAndServe("0.0.0.0:8000")
	if err != nil {
		log.Println(err)
	}
}

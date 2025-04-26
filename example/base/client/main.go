package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"log"
	"os"
)

func main() {
	// 创建一个节点为8081的节点
	c := node.NewClientOption(8081, 8000)
	// OnMessage 注册接收消息OnMessage回调函数，同步调用
	c.OnMessage(func(r responsewriter.ResponseWriter, m *message.Message) {
		log.Println("OnMessage", m.String())
		r.Response(message.StateCode_Success, append([]byte(fmt.Sprintf("response from %d: ", c.NodeId())), m.Data...))
	})
	// OnClose 注册OnClose回调函数，连接被关闭后的回调函数
	c.OnClose(func(err error) {
		log.Println("OnClose", err)
	})
	err := c.Connect("0.0.0.0:8000")
	if err != nil {
		log.Fatalln(err)
	}
	defer c.Close()
	scan := bufio.NewScanner(os.Stdin)
	print(">>")
	for scan.Scan() {
		switch scan.Text() {
		case "":
		case "exit":
			print("bye~")
			return
		default:
			res, code, err := c.Request(context.Background(), scan.Bytes())
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(code, string(res))
			}
		}
		print(">>")
	}
}

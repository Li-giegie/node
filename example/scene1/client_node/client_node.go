package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"log"
	"os"
	"time"
)

func main() {
	// 发起连接
	client, err := node.DialTCP(0, "0.0.0.0:8080", 1)
	if err != nil {
		log.Fatal(err)
	}
	// 添加处理方法
	client.HandleFunc(1, func(ctx *common.Context) {
		log.Println("receive", ctx.String())
		ctx.Write([]byte("ok"))
	})
	// 认证连接
	conn, err := client.AuthenticationWithServer(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	// 心跳包
	err = conn.Tick(time.Second, time.Second*3, time.Second*10, true)
	if err != nil {
		fmt.Print(err)
		return
	}
	go Request(conn)
	// 开启服务
	err = conn.Serve()
	if err != nil {
		log.Fatal(err)
	}
}

func Request(conn common.Conn) {
	stdin := bufio.NewScanner(os.Stdin)
	fmt.Print(">> ")
	for stdin.Scan() {
		if conn.State() != common.ConnStateTypeOnConnect {
			log.Fatalln("disconnect")
		}
		if len(stdin.Bytes()) == 0 {
			fmt.Print(">> ")
			continue
		}
		log.Println("send: ", stdin.Text())
		resp, err := conn.Request(context.Background(), 1, stdin.Bytes())
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(string(resp))
		fmt.Print(">> ")
	}
}

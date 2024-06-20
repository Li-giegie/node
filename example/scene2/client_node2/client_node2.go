package main

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"log"
	"time"
)

func main() {
	client, err := node.DialTCP(0, "0.0.0.0:8080", 2)
	if err != nil {
		log.Fatal(err)
	}
	client.HandleFunc(1, func(ctx *common.Context) {
		ctx.Write([]byte(fmt.Sprintf("%s client 2 response ok", time.Now().Format("2006/01/02 15:04:05"))))
	})
	conn, err := client.AuthenticationWithServer(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	go func() {
		go func() {
			time.Sleep(time.Second * 5)
			conn.Close()
		}()
		time.Sleep(time.Second)
		log.Println("request server 1")
		resp, err := conn.Request(context.Background(), 0, nil)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(string(resp))
		log.Println("request server forward client 1")
		resp, err = conn.Request(context.Background(), 1, nil)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(string(resp))
		log.Println("forward client 1")
		resp, err = conn.Forward(context.Background(), 1, 1, []byte("hello"))
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(string(resp))

	}()
	err = conn.Serve()
	if err != nil {
		log.Fatal(err)
	}
}

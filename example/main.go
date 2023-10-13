package main

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"log"
	"time"
)

func main() {
	go Server()
	time.Sleep(time.Second)
	Client()
	time.Sleep(time.Second)
}

func Client() {
	client := node.NewClient(node.DEFAULT_ClientID, node.DEFAULT_ServerAddress)
	conn, err := client.Connect(nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	resp, err := conn.Request(context.Background(), 1, []byte("hello"))
	if err != nil {
		log.Println(err)
	}
	fmt.Println(resp.String())

	resp, err = conn.Request(context.Background(), 2, []byte("hello"))
	if err != nil {
		log.Println(err)
	}
	fmt.Println(resp.String())

	resp, err = conn.RequestForward(context.Background(), "remote-client", 2, []byte("hello"))
	if err != nil {
		log.Println(err)
	}
	fmt.Println(resp.String())
}

func Server() {
	srv := node.NewServer(node.DEFAULT_ServerID, node.DEFAULT_ServerAddress)
	defer srv.Close()
	srv.AuthenticationFunc = func(id string, data []byte) (ok bool, reply []byte) {
		if id != node.DEFAULT_ClientID {
			return false, []byte("unauthorized")
		}
		return true, []byte("ok")
	}
	srv.RouteManager.AddRoute(1, func(ctx *node.Context) {
		fmt.Println(ctx.String())
		ctx.Write([]byte("收到"))
	})

	if err := srv.ListenAndServer(); err != nil {
		log.Println(err)
	}
}

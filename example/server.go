package main

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"log"
)

func Server() {
	srv := node.NewServer(node.DEFAULT_ServerAddress,
		node.WithSrvId(node.DEFAULT_ServerID),
		node.WithSrvConnectionEnableFunc(func(conn node.Conn) {
			log.Println("new connect ---", conn.Id())
		}),
	)
	srv.HandleFunc(1, func(id uint64, data []byte) (out []byte, err error) {
		resp, err := srv.Request(context.Background(), 379, 100, []byte("i'm server,I'm looking for the 379  \n"))
		if err != nil {
			return resp, err
		}
		re := append(append(data, []byte("i'm server\n")...), resp...)
		return re, nil
	})
	defer srv.Shutdown()
	if err := srv.ListenAndServer(); err != nil {
		log.Fatalln(err)
	}
}

func Client() {
	client := node.NewClient(node.DEFAULT_ServerAddress)
	_, err := client.Connect()
	if err != nil {
		log.Println(err)
		return
	}
	defer client.Close(true)
	resp, err := client.Request(context.Background(), 1, []byte("hello server, i'm client 20230\n"))
	if err != nil {
		log.Println("err 1 ", err)
		return
	}
	fmt.Println("req:", string(resp))
}

func HandlerClient() {
	client := node.NewClient(node.DEFAULT_ServerAddress, node.WithClientLocalIpAddr("0.0.0.0:3790"), node.WithClientId(379))
	_, err := client.Connect()
	if err != nil {
		log.Println(err)
		return
	}
	defer client.Close(true)
	client.HandleFunc(100, func(id uint64, data []byte) (out []byte, err error) {
		return []byte("i'm client 379 handle success\n"), nil
	})

	client.Run()
}

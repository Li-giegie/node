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
	go ForwardClient()
	time.Sleep(time.Second)
	go Client()
	time.Sleep(time.Second)
	select {}
}

func Client() {
	client := node.NewClient(node.DEFAULT_ServerAddress)
	_, err := client.Connect(nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer client.Close()
	resp, err := client.Request(context.Background(), 1, []byte("hello server, i'm client 20230\n"))
	if err != nil {
		log.Println("err 1 ", err)
		return
	}
	fmt.Println("req:", string(resp))
	//resp, err = client.RequestForward(context.Background(), 379, 100, []byte("hello"))
	//if err != nil {
	//	log.Println("err 3 ", err)
	//}
	//fmt.Println("forward ", string(resp))
}

func ForwardClient() {
	client := node.NewClient(node.DEFAULT_ServerAddress, node.WithClientLocalIpAddr("0.0.0.0:3790"), node.WithClientId(379))
	_, err := client.Connect(nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer client.Close()
	client.AddRoute(100, func(id uint64, data []byte) (out []byte, err error) {
		return []byte("i'm client 379 handle success\n"), nil
	})
	client.Run()
}

func Server() {
	srv, err := node.NewServer(node.DEFAULT_ServerAddress,
		node.WithSrvId(node.DEFAULT_ClientID),
	)
	srv.AuthenticationFunc = func(id uint64, data []byte) (ok bool, reply []byte) {
		return true, nil
	}
	srv.AddRoute(1, func(id uint64, data []byte) (out []byte, err error) {
		resp, err := srv.Request(context.Background(), 379, 100, []byte("i'm server,I'm looking for the 379  \n"))
		if err != nil {
			return resp, err
		}
		re := append(append(data, []byte("i'm server\n")...), resp...)
		return re, nil
	})
	defer srv.Shutdown()
	if err = srv.ListenAndServer(); err != nil {
		log.Fatalln(err)
	}

}

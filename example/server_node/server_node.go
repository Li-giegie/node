package main

import (
	"fmt"
	"github.com/Li-giegie/node"
	"log"
)

const srvAddr = "0.0.0.0:8080"

func main() {
	serverNode()
}

func serverNode() {
	srv := node.NewServer(srvAddr)
	srv.HandleFunc(1000, func(ctx *node.Context) {
		fmt.Println("1000 handle: ", string(ctx.Data()))
	})
	srv.HandleFunc(1001, func(ctx *node.Context) {
		fmt.Println("1001 handle: ", string(ctx.Data()))
		ctx.Reply(append([]byte("roger that:"), ctx.Data()...))
	})
	if err := srv.ListenAndServer(true); err != nil {
		log.Println(err)
	}
}

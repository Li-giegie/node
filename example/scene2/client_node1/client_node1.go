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
	client, err := node.DialTCP(0, "0.0.0.0:8080", 1)
	if err != nil {
		log.Fatal(err)
	}
	client.HandleFunc(1, func(ctx *common.Context) {
		ctx.Write([]byte(fmt.Sprintf("%s client 1 response ok", time.Now().Format("2006/01/02 15:04:05"))))
	})
	conn, err := client.AuthenticationWithServer(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	err = conn.Serve()
	if err != nil {
		log.Fatal(err)
	}
}

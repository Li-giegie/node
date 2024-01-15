package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"log"
	"time"
)

const (
	clientReqApi  = 100
	clientSendApi = 101
)
const serverAddr = "39.101.193.248:8088"

var srvaddr = flag.String("rip", serverAddr, "remote ip")

func main() {
	flag.Parse()
	Client("0.0.0.0:8989", *srvaddr, 1)
}

func Client(lAddr, rAddr string, id uint64) {
	client := node.NewClient(
		rAddr,
		node.WithClientId(id),
		node.WithClientLocalIpAddr(lAddr),
		node.WithClientKeepAlive(time.Second*3),
	)
	_, err := client.Connect(node.DEFAULT_ServerID, []byte{})
	if err != nil {
		log.Println(err)
		return
	}
	defer client.Close(true)
	client.HandleFunc(10, func(ctx *node.Context) {
		fmt.Println("receive msg with handle 10: ", ctx.String())
	})

	client.HandleFunc(20, func(ctx *node.Context) {
		fmt.Println("receive msg with handle 20: ", ctx.String())
		rep := append([]byte("handle 20 success: "), ctx.Data()...)
		_ = ctx.Reply(rep)
	})
	client.HandleFunc(3, func(ctx *node.Context) {
		fmt.Println("receive msg with handle 30: ", ctx.String())
		_ = ctx.ReplyErr(errors.New("err: test error reply"), append([]byte("handle 30 error: "), ctx.Data()...))
	})
	badApi, err := client.Registration(3)
	if err != nil {
		log.Println(err, badApi)
		return
	}
	if err = client.Run(); err != nil {
		log.Println(err)
	}
}

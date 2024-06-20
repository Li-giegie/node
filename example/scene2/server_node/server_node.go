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
	srv, err := node.ListenTCP(0, "0.0.0.0:8080")
	if err != nil {
		log.Fatal(err)
	}
	srv.HandleFunc(0, func(ctx *common.Context) {
		ctx.Write([]byte(fmt.Sprintf("%s server response %s", time.Now().Format("2006/01/02 15:04:05"), ctx.Data())))
	})
	//forward to client 1
	srv.HandleFunc(1, func(ctx *common.Context) {
		conn, ok := srv.GetConn(1)
		if !ok {
			ctx.Write([]byte("server forward err client 1 not exist"))
			return
		}
		resp, err := conn.Request(context.Background(), 1, ctx.Data())
		if err != nil {
			ctx.Write([]byte("server forward err client 1 " + err.Error()))
			return
		}
		ctx.Write(resp)
	})
	err = srv.Serve()
	if err != nil {
		log.Fatal(err)
	}

}

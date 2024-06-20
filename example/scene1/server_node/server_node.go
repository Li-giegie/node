package main

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"log"
	"time"
)

func main() {
	// 开启侦听
	srv, err := node.ListenTCP(0, "0.0.0.0:8080")
	if err != nil {
		return
	}
	// 绑定处理函数
	srv.HandleFunc(1, func(ctx *common.Context) {
		log.Println("receive: ", ctx.String())
		rData := []byte(fmt.Sprintf("%s reply ", time.Now().Format("2006/01/02 15:04:05")))
		rData = append(rData, ctx.Data()...)
		ctx.Write(rData)
	})
	// 开启服务
	err = srv.Serve()
	if err != nil {
		log.Fatalln(err)
	}
}

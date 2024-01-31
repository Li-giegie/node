package main

import (
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/example"
	"log"
	"time"
)

func main() {
	clientNode(example.SERVER_ADDR)
}

func clientNode(addr string) {
	cli := node.NewClient(addr,
		node.WithClientId(example.CLIENT1_ID),
		node.WithClientKeepAlive(node.DEFAULT_ConnectionIdle),
		node.WithClientLocalIpAddr(example.CLIENT1_ADDR),
	)
	//发起连接：入参dstId：目的Id即server id，authData 认证发送的数据，authReply 服务端认证回复 err 如果为空表示连接建立成功
	reply, err := cli.Connect(example.SERVER_ID, []byte("permit"))
	if err != nil {
		panic(err)
	}
	defer cli.Close(true)
	log.Printf("Connect reply: %s\n", reply)
	//仅发送，不会等待回复
	err = cli.Send(example.SERVER_SEND_API, []byte("head shot ~"))
	if err != nil {
		panic(err)
	}
	//发送并等待回复
	reply, err = cli.Request(time.Second*3, example.SERVER_REQUEST_API, []byte("stick together team ~"))
	if err != nil {
		panic(err)
	}
	log.Printf("%s\n", reply)
}

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
		node.WithClientId(example.CLIENT3_ID),
		node.WithClientLocalIpAddr(example.CLIENT3_ADDR),
	)
	//发起连接：入参dstId：目的Id即server id，authData 认证发送的数据，authReply 服务端认证回复 err 如果为空表示连接建立成功
	reply, err := cli.Connect(node.DEFAULT_ServerID, []byte("permit"))
	if err != nil {
		panic(err)
	}
	defer cli.Close(true)
	log.Printf("%s\n", reply)
	reply, err = cli.Request(time.Second*3, example.SERVER_FORWARD_API, []byte("stick together team ~"))
	if err != nil {
		panic(err)
	}
	log.Println("Request reply: ", string(reply))
}

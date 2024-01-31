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
		node.WithClientId(example.CLIENT2_ID),
		node.WithClientLocalIpAddr(example.CLIENT2_ADDR),
	)
	//发起连接：入参dstId：目的Id即server id，authData 认证发送的数据，authReply 服务端认证回复 err 如果为空表示连接建立成功
	reply, err := cli.Connect(example.SERVER_ID, []byte("permit"))
	if err != nil {
		panic(err)
	}
	defer cli.Close(true)
	log.Printf("%s\n", reply)

	//req c2 --> s --> c1 ; res c1 --> s --> c2
	reply, err = cli.Forward(time.Second*3, example.CLIENT1_ID, example.SERVER_FORWARD_API, []byte("client_node3: forward test ----"))
	if err != nil {
		panic(err)
	}
	log.Println("forward reply: ", string(reply))
	//req c2 --> s -->s handle  --> c1 ; res c1 -- s  --> s handle --> --> c2
	reply, err = cli.Request(time.Second*3, example.SERVER_FORWARD_API, []byte("stick together team ~"))
	if err != nil {
		panic(err)
	}
	log.Println("Request reply: ", string(reply))
}

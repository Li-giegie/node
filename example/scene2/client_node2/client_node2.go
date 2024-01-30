package main

import (
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/example/scene2"
	"log"
	"time"
)

func main() {
	clientNode(scene2.SERVER_ADDR)
}

func clientNode(addr string) {
	cli := node.NewClient(addr,
		node.WithClientId(scene2.CLIENT2_ID),
		node.WithClientLocalIpAddr(scene2.CLIENT2_ADDR),
	)
	//发起连接：入参dstId：目的Id即server id，authData 认证发送的数据，authReply 服务端认证回复 err 如果为空表示连接建立成功
	reply, err := cli.Connect(node.DEFAULT_ServerID, []byte("permit"))
	if err != nil {
		panic(err)
	}
	defer cli.Close(true)
	log.Printf("%s\n", reply)

	//req c2 --> s -->s --> c1 ; res c1 --> c2
	reply, err = cli.Forward(time.Second*3, scene2.CLIENT1_ID, scene2.SERVER_FORWARD_API, []byte("client_node3: forward test ----"))
	if err != nil {
		panic(err)
	}
	log.Println("forward reply: ", string(reply))
	//req c2 --> s -->s handle  --> c1 ; res c1 --> c2
	reply, err = cli.Request(time.Second*3, scene2.SERVER_FORWARD_API, []byte("stick together team ~"))
	if err != nil {
		panic(err)
	}
	log.Println("Request reply: ", string(reply))
}

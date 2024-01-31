package main

import (
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/example"
	"log"
)

func main() {
	clientNode(example.SERVER_ADDR)
}

func clientNode(addr string) {
	cli := node.NewClient(addr,
		node.WithClientId(example.CLIENT1_ID),
		node.WithClientLocalIpAddr(example.CLIENT1_ADDR),
	)
	//发起连接：入参dstId：目的Id即server id，authData 认证发送的数据，authReply 服务端认证回复 err 如果为空表示连接建立成功
	reply, err := cli.Connect(example.SERVER_ID, []byte("permit"))
	if err != nil {
		panic(err)
	}
	defer cli.Close(true)
	log.Printf("%s\n", reply)

	cli.HandleFunc(example.SERVER_FORWARD_API, func(ctx *node.Context) {
		// todo: ......
		_ = ctx.Reply([]byte("client_node1 handle success"))
	})
	if err = cli.Run(); err != nil {
		panic(err)
	}
}

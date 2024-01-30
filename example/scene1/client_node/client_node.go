package main

import (
	"flag"
	"github.com/Li-giegie/node"
	"log"
	"time"
)

var srvAddr = flag.String("addr", "0.0.0.0:8080", "ip:port")

func main() {
	flag.Parse()
	clientNode(*srvAddr)
}

func clientNode(addr string) {
	cli := node.NewClient(addr,
		node.WithClientId(node.DEFAULT_ClientID),
		node.WithClientKeepAlive(node.DEFAULT_ConnectionIdle),
		node.WithClientLocalIpAddr("0.0.0.0:8888"),
	)
	//发起连接：入参dstId：目的Id即server id，authData 认证发送的数据，authReply 服务端认证回复 err 如果为空表示连接建立成功
	reply, err := cli.Connect(node.DEFAULT_ServerID, []byte("permit"))
	if err != nil {
		panic(err)
	}
	defer cli.Close(true)
	log.Printf("%s\n", reply)
	//仅发送，不会等待回复
	err = cli.Send(1000, []byte("head shot ~"))
	if err != nil {
		panic(err)
	}
	//发送并等待回复
	reply, err = cli.Request(time.Second*3, 1001, []byte("stick together team ~"))
	if err != nil {
		panic(err)
	}
	log.Printf("%s\n", reply)
}

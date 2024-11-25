package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/example/bridge/client/cmd"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/protocol"
	"log"
	"net"
	"os"
	"time"
)

var lId = flag.Uint("lid", 0, "local id")
var rId = flag.Uint("rid", 0, "remote id")
var rAddr = flag.String("raddr", "", "remote address")
var rKey = flag.String("rkey", "hello", "remote key")
var timeout = flag.Duration("timeout", time.Second*6, "remote auth timeout")

func main() {
	flag.Parse()
	exitC := make(chan struct{}, 1)
	c := node.NewClient(uint32(*lId), &node.Identity{Id: uint32(*rId), Key: []byte(*rKey), Timeout: *timeout}, nil)
	// hello 协议用于连接保活
	hello := protocol.NewHelloProtocol(c, time.Second, time.Second*5, time.Second*30)
	defer hello.Stop()
	c.AddOnConnection(func(conn iface.Conn) {
		fmt.Println(conn.RemoteId())
	})
	c.AddOnMessage(func(ctx iface.Context) {
		fmt.Println(ctx.String())
		data := fmt.Sprintf("from %d echo %s", c.Id(), ctx.Data())
		ctx.Reply([]byte(data))
	})
	c.AddOnClosed(func(conn iface.Conn, err error) {
		exitC <- struct{}{}
	})
	netConn, err := net.Dial("tcp", *rAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	conn, err := c.Start(netConn)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()
	// 命令解析处理
	go handle(conn)
	<-exitC
}

func handle(conn iface.Conn) {
	envName := fmt.Sprintf("%d@>>", conn.LocalId())
	ctx := context.WithValue(context.Background(), "conn", conn)
	s := bufio.NewScanner(os.Stdin)
	fmt.Print(envName)
	for s.Scan() {
		if len(s.Bytes()) == 0 {
			fmt.Print(envName)
			continue
		}
		_, err := cmd.Group.ExecuteCmdLineContext(ctx, s.Text())
		if err != nil {
			log.Println(err)
		}
		fmt.Print(envName)
	}
}

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/example/bridge/client/cmd"
	"github.com/Li-giegie/node/pkg/common"
	"github.com/Li-giegie/node/pkg/conn"
	context2 "github.com/Li-giegie/node/pkg/ctx"
	"log"
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
	c := node.NewClient(uint32(*lId), &common.Identity{Id: uint32(*rId), Key: []byte(*rKey), AuthTimeout: *timeout})
	c.OnConnect(func(conn conn.Conn) {
		fmt.Println(conn.RemoteId())
	})
	c.OnMessage(func(ctx context2.Context) {
		fmt.Println(ctx.String())
		data := fmt.Sprintf("from %d echo %s", c.Id(), ctx.Data())
		ctx.Response(200, []byte(data))
	})
	c.OnClose(func(conn conn.Conn, err error) {
		exitC <- struct{}{}
	})
	log.Println("Connect addr", *rAddr)
	err := c.Connect(*rAddr)
	if err != nil {
		log.Fatalln(err)
	}
	defer c.Close()
	// 命令解析处理
	go handle(c)
	<-exitC
}

func handle(conn conn.Conn) {
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

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/example/bridge/client/cmd"
	"github.com/Li-giegie/node/pkg/client"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
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
	c := node.NewClientOption(uint32(*lId), uint32(*rId),
		client.WithRemoteKey([]byte(*rKey)),
		client.WithAuthTimeout(*timeout),
	)
	c.OnMessage(func(r responsewriter.ResponseWriter, m *message.Message) (next bool) {
		fmt.Printf("request from %d: %s\n", m.SrcId, m.Data)
		r.Response(message.StateCode_Success, []byte(fmt.Sprintf("response from %d: ok", c.NodeId())))
		return false
	})
	c.OnClose(func(conn conn.Conn, err error) (next bool) {
		fmt.Println("client close")
		exitC <- struct{}{}
		return true
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

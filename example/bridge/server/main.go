package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/example/bridge/server/cmd"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	"github.com/Li-giegie/node/protocol"
	"github.com/Li-giegie/node/protocol/routerbfs"
	"log"
	"net"
	"os"
	"time"
)

var id = flag.Uint("id", 8000, "local server id")
var addr = flag.String("addr", "0.0.0.0:8000", "local server address")
var key = flag.String("key", "hello", "local server key")
var timeout = flag.Duration("timeout", time.Second*6, "local server auth timeout")

func main() {
	flag.Parse()
	// 创建Server
	s := node.NewServer(&node.Identity{
		Id:          uint32(*id),
		Key:         []byte(*key),
		AuthTimeout: *timeout,
	})
	h := node.NewHandler(s)
	h.OnAccept(func(conn net.Conn) (allow bool) {
		log.Println("OnAccept remote addr:", conn.RemoteAddr())
		return true
	})
	h.OnConnect(func(conn iface.Conn) {
		log.Println("OnConnect remote id:", conn.RemoteId())
	})
	h.OnMessage(func(ctx iface.Context) {
		log.Println("OnMessage", ctx.String())
		ctx.Response(message.StateCode_Success, []byte(fmt.Sprintf("from %d response %s", s.Id(), ctx.Data())))
	})
	h.OnClose(func(conn iface.Conn, err error) {
		log.Println("OnClose", conn.RemoteId(), err)
	})
	helloProtocol := protocol.NewHelloProtocol(time.Second*5, time.Second*15, time.Second*45)
	h.Register(helloProtocol.ProtocolType(), helloProtocol)
	defer helloProtocol.Stop()

	//开启节点发现协议
	bfsProtocol := protocol.NewRouterBFSProtocol(s)
	h.Register(bfsProtocol.ProtocolType(), bfsProtocol)
	// 解析命令
	go handle(s, nil)
	log.Println("Listen on", *addr)
	if err := s.ListenAndServe(*addr); err != nil {
		log.Println(err)
	}
}

func handle(s iface.Server, p routerbfs.Protocol) {
	ctx := context.WithValue(context.WithValue(context.Background(), "server", s), "bfs", p)
	time.Sleep(time.Second)
	envName := fmt.Sprintf("%d@>>", *id)
	sc := bufio.NewScanner(os.Stdin)
	fmt.Print(envName)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			fmt.Print(envName)
			continue
		}
		executeCmd, err := cmd.Group.ExecuteCmdLineContext(ctx, sc.Text())
		if err != nil {
			if executeCmd == nil {
				cmd.Group.Usage()
			} else {
				fmt.Println(err)
			}
		}
		fmt.Print(envName)
	}
}

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/example/bridge/server/cmd"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/protocol"
	"github.com/Li-giegie/node/protocol/hello"
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
		Id:      uint32(*id),
		Key:     []byte(*key),
		Timeout: *timeout,
	}, nil)
	s.AddOnConnect(func(conn iface.Conn) {
		log.Println("connection", conn.RemoteId())
	})
	s.AddOnMessage(func(ctx iface.Context) {
		fmt.Println(ctx.String())
		data := fmt.Sprintf("from %d echo %s", s.Id(), ctx.Data())
		ctx.Reply([]byte(data))
	})
	// 开启hello协议
	{
		HP := protocol.NewHelloProtocol(s, time.Second*5, time.Second*15, time.Second*45)
		// 收集hello协议的事件
		HP.SetEventCallback(func(action hello.Event_Action, val interface{}) {
			fmt.Println(action.String(), val)
		})
		defer HP.Stop()
	}
	// 开启节点发现协议
	NDP := protocol.NewNodeDiscoveryProtocol(s)
	l, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	ctx := context.WithValue(context.WithValue(context.Background(), "server", s), "ndp", NDP)
	// 解析命令
	go handle(ctx)
	log.Println("start success", *addr)
	if err = s.Serve(l); err != nil {
		fmt.Println(err)
	}
}

func handle(ctx context.Context) {
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

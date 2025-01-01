package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/example/bridge/server/cmd"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/handler"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/protocol"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"github.com/Li-giegie/node/pkg/server"
	"log"
	"os"
	"time"
)

var id = flag.Uint("id", 8000, "local server id")
var addr = flag.String("addr", "0.0.0.0:8000", "local server address")
var key = flag.String("key", "hello", "local server key")
var timeout = flag.Duration("timeout", time.Second*6, "local server auth timeout")

func main() {
	flag.Parse()
	s := node.NewServerOption(uint32(*id),
		server.WithAuthKey([]byte(*key)),
		server.WithAuthTimeout(*timeout),
	)
	s.OnClose(func(conn conn.Conn, err error) (next bool) {
		fmt.Println("on close", conn.RemoteId())
		return true
	})
	//开启节点发现协议
	bfsProtocol := protocol.NewRouterBFSProtocol(s)
	s.Register(bfsProtocol.ProtocolType(), bfsProtocol)
	s.Register(message.MsgType_Default, &handler.Default{OnMessageFunc: func(r responsewriter.ResponseWriter, m *message.Message) {
		fmt.Printf("request from %d: %s\n", m.SrcId, m.Data)
		r.Response(message.StateCode_Success, []byte(fmt.Sprintf("response from %d: ok", s.NodeId())))
	}})
	// 解析命令
	go handle(s, nil)
	log.Println("Listen on", *addr)
	if err := s.ListenAndServe(*addr); err != nil {
		log.Println(err)
	}
}

func handle(s server.Server, p protocol.Protocol) {
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

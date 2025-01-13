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
	"github.com/Li-giegie/node/pkg/protocol/routerbfs"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"github.com/Li-giegie/node/pkg/server"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var configFile = flag.String("c", "./1.json", "config file path")
var gen = flag.String("gen", "", "gen config file")

func init() {
	flag.Parse()
	if *gen != "" {
		genConfTemplate(*gen)
	} else {
		loadConfTemplate(*configFile)
	}
}

var bfsProtocol protocol.Protocol

func main() {
	s := node.NewServerOption(conf.Id,
		server.WithAuthKey([]byte(conf.Key)),
		server.WithAuthTimeout(time.Second*6),
	)
	s.OnClose(func(conn conn.Conn, err error) (next bool) {
		fmt.Println("on close", conn.RemoteId())
		return true
	})
	//开启节点发现协议
	bfsProtocol = protocol.NewRouterBFSProtocol(s)
	bfsProtocol.StartNodeSync(context.TODO(), time.Second*5)
	s.Register(bfsProtocol.ProtocolType(), bfsProtocol)
	s.Register(message.MsgType_Default, &handler.Default{OnMessageFunc: func(r responsewriter.ResponseWriter, m *message.Message) {
		if string(m.Data) == "exit" {
			for _, c := range s.GetAllConn() {
				_ = c.Send(m.Data)
			}
			s.Close()
			time.Sleep(time.Second)
			os.Exit(0)
		}
		fmt.Printf("request from %d: %s\n", m.SrcId, m.Data)
		r.Response(message.StateCode_Success, []byte(fmt.Sprintf("response from %d: ok", s.NodeId())))
	}})
	// 解析命令
	go handle(s, nil)
	go func() {
		for {
			time.Sleep(time.Second * 2)
			success := false
			run := false
			for !success {
				time.Sleep(time.Second * 2)
				success = true
				for _, bridgeNode := range conf.Bridge {
					if _, ok := s.GetConn(bridgeNode.Id); ok {
						continue
					}
					run = true
					conn, err := net.Dial("tcp", bridgeNode.Address)
					if err != nil {
						//log.Println(err)
						success = false
						continue
					}
					err = s.Bridge(conn, bridgeNode.Id, []byte("hello"))
					if err != nil {
						//log.Println(err)
						success = false
					}
				}
			}
			if run && success {
				for _, bridge := range conf.Bridge {
					fmt.Println("bridge", bridge.Id, bridge.Address)
				}
				fmt.Println("bridge success")
			}
		}
	}()
	log.Println("Listen on", conf.Addr)
	if err := s.ListenAndServe(conf.Addr); err != nil {
		log.Println(err)
	}
}

func handle(s server.Server, p protocol.Protocol) {
	ctx := context.WithValue(context.WithValue(context.Background(), "server", s), "bfs", p)
	time.Sleep(time.Second)
	envName := fmt.Sprintf("%d@>>", conf.Id)
	sc := bufio.NewScanner(os.Stdin)
	r := bfsProtocol.(*routerbfs.RouterBFS)
	fmt.Print(envName)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 {
			fmt.Print(envName)
			continue
		}
		switch fields[0] {
		case "exit":
			for _, c := range s.GetAllConn() {
				c.Send([]byte("exit"))
			}
			s.Close()
			time.Sleep(time.Second)
			os.Exit(0)
		case "full":
			for _, info := range r.FullNode.GetAllNodeInfo() {
				fmt.Print("root", info.RootId)
				for _, id := range info.SubIds {
					fmt.Print(" sub ", id.Id)
				}
				fmt.Println()
			}
		case "add", "remove":
			if len(fields) < 3 {
				fmt.Println("args < 4: add 1 2")
				fmt.Print(envName)
				continue
			}
			rid, err := strconv.Atoi(fields[1])
			sid, err2 := strconv.Atoi(fields[2])
			if err != nil || err2 != nil {
				fmt.Println(err, err2)
			} else {
				ok := true
				if fields[0] == "add" {
					ok = r.FullNode.Add(uint32(rid), uint32(sid), time.Now().UnixNano())
				} else {
					ok = r.FullNode.Remove(uint32(rid), uint32(sid), time.Now().UnixNano())
				}
				if ok {
					fmt.Printf("%s 操作成功 root %d sub %d\n", fields[0], rid, sid)
				} else {
					fmt.Printf("%s 操作失败 root %d sub %d\n", fields[0], rid, sid)
				}
			}
		default:
			executeCmd, err := cmd.Group.ExecuteCmdLineContext(ctx, sc.Text())
			if err != nil {
				if executeCmd == nil {
					cmd.Group.Usage()
				} else {
					fmt.Println(err)
				}
			}
		}
		fmt.Print(envName)
	}
}

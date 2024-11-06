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
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net"
	"os"
	"time"
)

type Conf struct {
	Addr                        string
	Id                          uint32
	EnableNodeDiscoveryProtocol bool
	HelloProtocol
}

type HelloProtocol struct {
	Enable       bool
	EnableStdout bool
	Interval     time.Duration
	Timeout      time.Duration
	TimeoutClose time.Duration
}

var confFile = flag.String("c", "./conf8000.yaml", "configure file")
var genConfFile = flag.String("gen", "", "gen config file template")

var c *Conf

func init() {
	flag.Parse()
	if *genConfFile != "" {
		data, err := yaml.Marshal(&Conf{
			Id:                          10,
			Addr:                        "0.0.0.0:8010",
			EnableNodeDiscoveryProtocol: true,
			HelloProtocol: HelloProtocol{
				Enable:       true,
				Interval:     time.Second * 5,
				Timeout:      time.Second * 10,
				TimeoutClose: time.Second * 30,
			},
		})
		if err != nil {
			log.Fatalln("gen err", err)
		}
		if err = os.WriteFile(*genConfFile, data, 0666); err != nil {
			log.Fatalln(err)
		}
		log.Println("gen success")
		os.Exit(0)
	}
	data, err := os.ReadFile(*confFile)
	if err != nil {
		log.Fatalln("read config err", err)
	}
	c = new(Conf)
	if err = yaml.Unmarshal(data, c); err != nil {
		log.Fatalln("Unmarshal conf err", err)
	}
}

func main() {
	// 开启侦听
	l, err := net.Listen("tcp", c.Addr)
	if err != nil {
		log.Fatalln(err)
	}
	// 创建Server
	s := node.NewServer(l, node.SrvConf{
		Identity: &node.Identity{
			Id:          c.Id,
			AuthKey:     []byte("hello"),
			AuthTimeout: time.Second * 3,
		},
		MaxMsgLen:          0xffffff,
		WriterQueueSize:    1024,
		ReaderBufSize:      4096,
		WriterBufSize:      4096,
		MaxConns:           0,
		MaxListenSleepTime: time.Minute,
		ListenStepTime:     time.Second,
	})
	s.AddOnConnection(func(conn iface.Conn) {
		log.Println("connection", conn.RemoteId(), conn.NodeType())
	})
	s.AddOnMessage(func(conn iface.Context) {
		fmt.Println(string(conn.Data()))
		data := fmt.Sprintf("from %d echo %s", s.Id(), conn.Data())
		conn.Reply([]byte(data))
	})
	if c.HelloProtocol.Enable {
		output := io.Discard
		if c.HelloProtocol.EnableStdout {
			output = os.Stdout
		}
		// 开启Hello 连接保活协议
		protocol.StartMultipleNodeHelloProtocol(context.Background(), s, s, c.Interval, c.Timeout, c.TimeoutClose, output)
	}
	if c.EnableNodeDiscoveryProtocol {
		// 开启节点路由动态发现协议
		protocol.StartDiscoveryProtocol(16, s, s)
	}
	// 解析命令
	go func() {
		time.Sleep(time.Second)
		sc := bufio.NewScanner(os.Stdin)
		ctx := context.WithValue(context.Background(), "server", s)
		fmt.Print(">>")
		for sc.Scan() {
			if len(sc.Bytes()) == 0 {
				fmt.Print(">>")
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
			fmt.Print(">>")
		}
	}()
	log.Println("start success", c.Addr)
	if err = s.Serve(); err != nil {
		println(err)
	}
}

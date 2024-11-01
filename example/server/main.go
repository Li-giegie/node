package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/example/server/cmd"
	"github.com/Li-giegie/node/protocol"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net"
	"os"
	"time"
)

type Conf struct {
	Addr string
	*node.SrvConf
	NodeDiscoveryProtocol
	HelloProtocol
}

type NodeDiscoveryProtocol struct {
	Enable        bool
	QueryInterval time.Duration
}

type HelloProtocol struct {
	Enable       bool
	EnableStdout bool
	OutFile      string
	Interval     time.Duration
	Timeout      time.Duration
	TimeoutClose time.Duration
}

var confFile = flag.String("conf", "./conf8000.yaml", "configure file")
var genConfFile = flag.String("gen", "", "gen config file template")

var c *Conf

func init() {
	flag.Parse()
	if *genConfFile != "" {
		data, err := yaml.Marshal(&Conf{
			Addr: "0.0.0.0:8010",
			SrvConf: &node.SrvConf{
				Identity: &node.Identity{
					Id:          10,
					AuthKey:     []byte("hello"),
					AuthTimeout: time.Second * 6,
				},
				MaxMsgLen:          0xffffff,
				WriterQueueSize:    1024,
				ReaderBufSize:      4096,
				WriterBufSize:      4096,
				MaxConns:           0,
				MaxListenSleepTime: time.Minute,
				ListenStepTime:     time.Second,
			},
			NodeDiscoveryProtocol: NodeDiscoveryProtocol{
				Enable:        true,
				QueryInterval: time.Second * 30,
			},
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

type OnCustomMessageCallback func(ctx node.CustomContext) bool
type OnConnectionCallback func(ctx node.Conn)
type OnCloseCallback func(id uint32, err error)

func main() {
	l, err := net.Listen("tcp", c.Addr)
	if err != nil {
		log.Fatalln(err)
	}
	s := node.NewServer(l, &node.SrvConf{
		Identity:           c.Identity,
		MaxMsgLen:          0xffffff,
		WriterQueueSize:    1024,
		ReaderBufSize:      4096,
		WriterBufSize:      4096,
		MaxConns:           0,
		MaxListenSleepTime: time.Minute,
		ListenStepTime:     time.Second,
	})
	defer s.Close()
	var OnCustomMessage []OnCustomMessageCallback
	var OnConnection []OnConnectionCallback
	var OnClose []OnCloseCallback
	s.OnConnection = func(conn node.Conn) {
		for _, callback := range OnConnection {
			callback(conn)
		}
		log.Println("OnConnection", conn.RemoteId())
	}
	s.OnMessage = func(ctx node.Context) {
		log.Println("OnMessage", ctx.String())
		ctx.Reply(ctx.Data())
	}
	s.OnCustomMessage = func(ctx node.CustomContext) {
		for _, callback := range OnCustomMessage {
			if next := callback(ctx); !next {
				return
			}
		}
		log.Println("OnCustomMessage", ctx.String())
	}
	s.OnClose = func(id uint32, err error) {
		for _, callback := range OnClose {
			callback(id, err)
		}
		log.Println("OnClose", id, err)
	}
	if c.HelloProtocol.Enable {
		var w io.Writer
		if c.HelloProtocol.EnableStdout {
			w = os.Stdout
		}
		if c.HelloProtocol.OutFile != "" {
			f, err := os.OpenFile(c.OutFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer f.Close()
			if c.HelloProtocol.EnableStdout {
				w = io.MultiWriter(os.Stdout, f)
			}
		}
		h := protocol.NewHelloProtocol(c.HelloProtocol.Interval, c.HelloProtocol.Timeout, c.HelloProtocol.TimeoutClose, w)
		defer h.Stop()
		go h.KeepAliveMultiple(s.ConnManager)
		OnCustomMessage = append(OnCustomMessage, h.OnCustomMessage)
	}

	if c.NodeDiscoveryProtocol.Enable {
		nd := protocol.NewNodeDiscoveryProtocol(s.Id(), s.ConnManager, s.Router, os.Stdout)
		OnConnection = append(OnConnection, nd.Connection)
		OnCustomMessage = append(OnCustomMessage, nd.CustomHandle)
		OnClose = append(OnClose, nd.Disconnect)
	}
	// 解析命令
	go func() {
		time.Sleep(time.Second)
		sc := bufio.NewScanner(os.Stdin)
		fmt.Print(">>")
		for sc.Scan() {
			if sc.Text() != "" {
				executeCmd, err := cmd.Group.ExecuteCmdLineContext(context.WithValue(context.Background(), "server", s), sc.Text())
				if err != nil {
					if executeCmd == nil {
						cmd.Group.Usage()
					} else {
						fmt.Println(err)
					}
				}
			}
			fmt.Print(">>")
		}
	}()
	log.Println("start success", c.Addr)
	if err = s.Serve(); err != nil {
		println(err)
		return
	}
}

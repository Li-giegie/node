package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/example/client/cmd"
	"github.com/Li-giegie/node/protocol"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net"
	"os"
	"time"
)

type Config struct {
	RAddr string
	*node.Identity
	HelloProtocol
}

type HelloProtocol struct {
	Enable       bool
	EnableStdout bool
	OutFile      string
	Interval     time.Duration
	Timeout      time.Duration
	TimeoutClose time.Duration
}

var conf = flag.String("conf", "./conf8000.yaml", "configure file path")
var gen = flag.String("gen", "", "generate configure file template")

var c = new(Config)

func init() {
	flag.Parse()
	if *gen != "" {
		data, err := yaml.Marshal(&Config{
			RAddr: "0.0.0.0:8010",
			Identity: &node.Identity{
				Id:          8001,
				AuthKey:     []byte("hello"),
				AuthTimeout: time.Second * 6,
			},
			HelloProtocol: HelloProtocol{
				Enable:       true,
				EnableStdout: true,
				OutFile:      "",
				Interval:     time.Second * 3,
				Timeout:      time.Second * 15,
				TimeoutClose: time.Second * 30,
			},
		})
		if err != nil {
			log.Fatalln(err)
		}
		if err = os.WriteFile(*gen, data, 0666); err != nil {
			log.Fatalln(err)
		}
		log.Println("generate success")
		os.Exit(0)
	}
	data, err := os.ReadFile(*conf)
	if err != nil {
		log.Fatalln(err)
	}
	if err = yaml.Unmarshal(data, c); err != nil {
		log.Fatalln(err)
	}
}

type OnCustomMessageCallback func(ctx node.CustomContext) bool

func main() {
	conn, err := net.Dial("tcp", c.RAddr)
	if err != nil {
		log.Fatalln(err)
	}
	client := node.NewClient(conn, &node.CliConf{
		ReaderBufSize:   4096,
		WriterBufSize:   4096,
		WriterQueueSize: 1024,
		MaxMsgLen:       0xffffff,
		ClientIdentity: &node.ClientIdentity{
			Id:            c.Id,
			RemoteAuthKey: c.AuthKey,
			Timeout:       c.AuthTimeout,
		},
	})
	var OnCustomMessage []OnCustomMessageCallback
	client.OnConnection = func(conn node.Conn) {
		log.Println("OnConnection", conn.RemoteId())
	}
	client.OnMessage = func(ctx node.Context) {
		ctx.Reply(ctx.Data())
	}
	client.OnCustomMessage = func(ctx node.CustomContext) {
		for _, callback := range OnCustomMessage {
			if next := callback(ctx); !next {
				return
			}
		}
		log.Println("OnCustomMessage", ctx.String())
	}
	stopC := make(chan struct{})
	client.OnClose = func(id uint32, err error) {
		stopC <- struct{}{}
	}
	if err = client.Start(); err != nil {
		log.Fatalln(err)
	}
	//是否开启hello (心跳)协议
	if c.HelloProtocol.Enable {
		w := io.Discard
		if c.EnableStdout {
			w = os.Stdout
		}
		if c.OutFile != "" {
			f, err := os.OpenFile(c.OutFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
			if err != nil {
				log.Println(err)
				return
			}
			defer f.Close()
			if c.EnableStdout {
				w = io.MultiWriter(os.Stdout, f)
			} else {
				w = f
			}
		}
		h := protocol.NewHelloProtocol(c.Interval, c.Timeout, c.TimeoutClose, w)
		OnCustomMessage = append(OnCustomMessage, h.OnCustomMessage)
		go h.KeepAlive(client)
		defer h.Stop()
	}
	// 命令解析处理
	go func() {
		ctx2 := context.WithValue(context.Background(), "client", client)
		s := bufio.NewScanner(os.Stdin)
		fmt.Print(">>")
		for s.Scan() {
			if len(s.Text()) > 0 {
				_, err := cmd.Group.ExecuteCmdLineContext(ctx2, s.Text())
				if err != nil {
					log.Println(err)
				}
			}
			fmt.Print(">>")
		}
	}()
	<-stopC
}

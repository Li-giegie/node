package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/example/client/cmd"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/protocol"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net"
	"os"
	"time"
)

type Config struct {
	Id    uint32
	RAddr string
	HelloProtocol
}

type HelloProtocol struct {
	Enable       bool
	EnableStdout bool
	Interval     time.Duration
	Timeout      time.Duration
	TimeoutClose time.Duration
}

var confFile = flag.String("c", "./conf8000.yaml", "configure file path")
var gen = flag.String("gen", "", "generate configure file template")

var conf = new(Config)

func init() {
	flag.Parse()
	if *gen != "" {
		data, err := yaml.Marshal(&Config{
			RAddr: "0.0.0.0:8010",
			Id:    10,
			HelloProtocol: HelloProtocol{
				Enable:       true,
				EnableStdout: true,
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
	data, err := os.ReadFile(*confFile)
	if err != nil {
		log.Fatalln(err)
	}
	if err = yaml.Unmarshal(data, conf); err != nil {
		log.Fatalln(err)
	}
}

func main() {
	conn, err := net.Dial("tcp", conf.RAddr)
	if err != nil {
		log.Fatalln(err)
	}
	c := node.NewClient(conn, &node.CliConf{
		ReaderBufSize:   4096,
		WriterBufSize:   4096,
		WriterQueueSize: 1024,
		MaxMsgLen:       0xffffff,
		ClientIdentity: &node.ClientIdentity{
			Id:            conf.Id,
			RemoteAuthKey: []byte("hello"),
			Timeout:       time.Second * 6,
		},
	})
	c.AddOnMessage(func(conn iface.Context) {
		data := fmt.Sprintf("from %d echo %s", c.LocalId(), conn.Data())
		conn.Reply([]byte(data))
	})
	stopC := make(chan struct{}, 1)
	c.AddOnClosed(func(conn iface.Conn, err error) {
		stopC <- struct{}{}
	})
	if err = c.Start(); err != nil {
		log.Fatalln(err)
	}
	if conf.HelloProtocol.Enable {
		output := io.Discard
		if conf.EnableStdout {
			output = os.Stdout
		}
		protocol.StartHelloProtocol(context.Background(), c, c, conf.Interval, conf.Timeout, conf.TimeoutClose, output)
	}

	// 命令解析处理
	go func() {
		ctx := context.WithValue(context.Background(), "client", c)
		s := bufio.NewScanner(os.Stdin)
		fmt.Print(">>")
		for s.Scan() {
			if len(s.Bytes()) == 0 {
				fmt.Print(">>")
				continue
			}
			_, err := cmd.Group.ExecuteCmdLineContext(ctx, s.Text())
			if err != nil {
				log.Println(err)
			}
			fmt.Print(">>")
		}
	}()
	<-stopC
}

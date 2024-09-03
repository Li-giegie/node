package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/example/client/cmd"
	"github.com/Li-giegie/node/protocol"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"time"
)

type Config struct {
	RAddr string
	*node.Identity
	*HelloProtocol
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
			RAddr: "0.0.0.0:8000",
			Identity: &node.Identity{
				Id:            8001,
				AccessKey:     []byte("hello"),
				AccessTimeout: time.Second * 6,
			},
			HelloProtocol: &HelloProtocol{
				Enable:       false,
				EnableStdout: false,
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

func main() {
	handler := new(ClientHandler)
	handler.stopChan = make(chan error)
	//是否开启hello (心跳)协议
	if c.HelloProtocol != nil && c.HelloProtocol.Enable {
		var w io.Writer
		if c.HelloProtocol.EnableStdout {
			w = os.Stdout
		}
		if c.HelloProtocol.OutFile != "" {
			f, err := os.OpenFile(c.HelloProtocol.OutFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
			if err != nil {
				log.Fatalln(err)
			}
			defer f.Close()
			if c.EnableStdout {
				w = io.MultiWriter(os.Stdout, f)
			} else {
				w = f
			}
		}
		handler.HelloProtocol = protocol.NewHelloProtocol(c.HelloProtocol.Interval, c.HelloProtocol.Timeout, c.HelloProtocol.TimeoutClose, w)
	}

	conn, err := node.DialTCP(c.RAddr, c.Identity, handler)
	if err != nil {
		log.Println(err)
		return
	}
	handler.Conn = conn
	if c.HelloProtocol.Enable {
		go handler.HelloProtocol.KeepAlive(conn)
	}
	log.Printf("client %d start success\n", c.Identity.Id)
	defer conn.Close()
	// 命令解析处理
	go func() {
		ctx2 := context.WithValue(context.Background(), "client", conn)
		s := bufio.NewScanner(os.Stdin)
		fmt.Print(">>")
		for s.Scan() && conn.State() == common.ConnStateTypeOnConnect {
			if len(s.Text()) > 0 {
				_, err := cmd.Group.ExecuteCmdLineContext(ctx2, s.Text())
				if err != nil {
					log.Println(err)
				}
			}
			fmt.Print(">>")
		}
	}()
	if err = <-handler.stopChan; err != nil {
		fmt.Println(err)
	}
}

type ClientHandler struct {
	common.Conn
	protocol.HelloProtocol
	stopChan chan error
}

func (c *ClientHandler) Connection(conn common.Conn) {
	log.Println("Connection", conn.RemoteId())
}

func (c *ClientHandler) Handle(ctx common.Context) {
	log.Printf("ClientHandler Handle src [%d] %s\n", ctx.SrcId(), ctx.Data())
	ctx.Reply([]byte(fmt.Sprintf("ClientHandler [%d] handle reply: %s", ctx.DestId(), ctx.Data())))
}

func (c *ClientHandler) ErrHandle(msg common.ErrContext, err error) {
	log.Println("ClientHandler ErrHandle: ", msg.String(), err)
}

func (c *ClientHandler) CustomHandle(ctx common.CustomContext) {
	if c.HelloProtocol != nil && !c.HelloProtocol.CustomHandle(ctx) {
		return
	}
	log.Println("client CustomHandle: ", ctx.String())
}

func (c *ClientHandler) Disconnect(id uint16, err error) {
	c.stopChan <- err
}

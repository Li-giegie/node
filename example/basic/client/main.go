package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/example/basic/client/cmd"
	"github.com/Li-giegie/node/protocol"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type Config struct {
	LId         uint16
	RAddr       string
	RKey        string
	AuthTimeout time.Duration
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

var conf = flag.String("conf", "./conf.yaml", "configure file path")
var gen = flag.String("gen", "", "generate configure file template")

var c = new(Config)

func init() {
	flag.Parse()
	if *gen != "" {
		data, err := yaml.Marshal(&Config{
			LId:   0,
			RAddr: "0.0.0.0:8000",
			RKey:  "hello",
			HelloProtocol: &HelloProtocol{
				Enable:       false,
				EnableStdout: false,
				OutFile:      "",
				Interval:     0,
				Timeout:      0,
				TimeoutClose: 0,
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
	handler.ClientAuthProtocol = protocol.NewClientAuthProtocol(c.LId, c.RKey, c.AuthTimeout)
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
		handler.ClientHelloProtocol = protocol.NewClientHelloProtocol(c.HelloProtocol.Interval, c.HelloProtocol.Timeout, c.HelloProtocol.TimeoutClose, w)
	}
	conn, err := node.Dial("tcp", c.RAddr, c.LId, handler)
	if err != nil {
		log.Println(err)
		return
	}
	handler.Conn = conn
	log.Printf("client %d start success\n", c.LId)
	defer conn.Close()
	go func() {
		ctx := context.WithValue(context.Background(), "client", conn)
		s := bufio.NewScanner(os.Stdin)
		fmt.Print(">>")
		for s.Scan() && conn.State() == common.ConnStateTypeOnConnect {
			if len(s.Text()) > 0 {
				_, err := cmd.Group.ExecuteContext(ctx, strings.Fields(s.Text()))
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
	protocol.ClientAuthProtocol
	protocol.ClientHelloProtocol
	stopChan chan error
}

func (c *ClientHandler) Init(conn net.Conn) (remoteId uint16, err error) {
	return c.ClientAuthProtocol.Init(conn)
}

func (c *ClientHandler) Connection(conn common.Conn) {
	log.Println("Connection", conn.RemoteId())
}

func (c *ClientHandler) Handle(ctx common.Context) {
	log.Printf("ClientHandler Handle src [%d] %s\n", ctx.SrcId(), ctx.Data())
	ctx.Reply([]byte(fmt.Sprintf("ClientHandler [%d] handle reply: %s", ctx.DestId(), ctx.Data())))
}

func (c *ClientHandler) ErrHandle(msg *common.Message, err error) {
	log.Println("ClientHandler ErrHandle: ", msg.String(), err)
}

func (c *ClientHandler) CustomHandle(ctx common.Context) {
	if c.ClientHelloProtocol != nil && !c.ClientHelloProtocol.CustomHandle(ctx) {
		return
	}
	log.Println("client CustomHandle: ", ctx.String())
}

func (c *ClientHandler) Disconnect(id uint16, err error) {
	go func() {
		c.stopChan <- err
	}()
}

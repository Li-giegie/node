package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/example/server/cmd"
	"github.com/Li-giegie/node/protocol"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"time"
)

type Conf struct {
	Addr string
	*node.Identity
	*NodeDiscoveryProtocol
	*HelloProtocol
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
			Addr: "0.0.0.0:8000",
			Identity: &node.Identity{
				Id:            0,
				AccessKey:     []byte("hello"),
				AccessTimeout: time.Second * 6,
			},
			NodeDiscoveryProtocol: &NodeDiscoveryProtocol{
				Enable:        false,
				QueryInterval: time.Second * 30,
			},
			HelloProtocol: &HelloProtocol{
				Enable:       false,
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

type NodeDiscoveryConf struct {
	id uint16
	*node.Conns
	common.Router
}

func (n *NodeDiscoveryConf) Id() uint16 {
	return n.id
}

type ServerHandle struct {
	protocol.NodeDiscoveryProtocol
	protocol.HelloProtocol
	*node.Server
}

func (h *ServerHandle) Connection(conn common.Conn) {
	log.Println("connection", conn.RemoteId())
	if h.NodeDiscoveryProtocol != nil {
		h.NodeDiscoveryProtocol.Connection(conn)
	}
}

func (h *ServerHandle) Handle(ctx common.Context) {
	log.Printf("server [%d] Handl datae %s\n", ctx.SrcId(), ctx.Data())
	ctx.Reply([]byte(fmt.Sprintf("server [%d] handle reply: %s", h.Server.Id(), ctx.Data())))
}

func (h *ServerHandle) ErrHandle(msg common.ErrContext, err error) {
	log.Println("ErrHandle", msg.String())
}

func (h *ServerHandle) CustomHandle(ctx common.CustomContext) {
	if h.HelloProtocol != nil && !h.HelloProtocol.CustomHandle(ctx) {
		return
	}
	if h.NodeDiscoveryProtocol != nil && !h.NodeDiscoveryProtocol.CustomHandle(ctx) {
		return
	}
	log.Println("CustomHandle", ctx.String())
}

func (h *ServerHandle) Disconnect(id uint16, err error) {
	log.Println("Disconnect", id, err)
	if h.NodeDiscoveryProtocol != nil {
		h.NodeDiscoveryProtocol.Disconnect(id, err)
	}
}

func main() {
	srv, err := node.ListenTCP(c.Addr, c.Identity)
	if err != nil {
		log.Fatalln(err)
	}
	defer srv.Close()

	s := new(ServerHandle)
	s.Server = srv
	if c.HelloProtocol != nil && c.HelloProtocol.Enable {
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
			} else {
				w = f
			}
		}
		s.HelloProtocol = protocol.NewHelloProtocol(c.HelloProtocol.Interval, c.HelloProtocol.Timeout, c.HelloProtocol.TimeoutClose, w)
		go s.HelloProtocol.KeepAliveMultiple(srv.Conns)
	}

	if c.NodeDiscoveryProtocol != nil && c.NodeDiscoveryProtocol.Enable {
		s.NodeDiscoveryProtocol = protocol.NewNodeDiscoveryProtocol(&NodeDiscoveryConf{srv.Id(), srv.Conns, srv.Router}, os.Stdout)
		go s.NodeDiscoveryProtocol.StartTimingQueryEnableProtoNode(context.Background(), c.NodeDiscoveryProtocol.QueryInterval)
	}
	// 解析命令
	go func() {
		time.Sleep(time.Second)
		sc := bufio.NewScanner(os.Stdin)
		fmt.Print(">>")
		for sc.Scan() && srv.State == node.StateType_Listen {
			if sc.Text() != "" {
				executeCmd, err := cmd.Group.ExecuteCmdLineContext(context.WithValue(context.Background(), "server", srv), sc.Text())
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
	if err = srv.Serve(s); err != nil {
		println(err)
		return
	}
}

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/example/multiple-domain/cmd"
	"github.com/Li-giegie/node/protocol"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

type Conf struct {
	Id              uint16
	Addr            string
	InitConnTimeout time.Duration
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

var confFile = flag.String("conf", "./conf.yaml", "configure file")
var genConfFile = flag.String("gen", "", "gen config file template")

func init() {
	flag.Parse()
	if *genConfFile != "" {
		data, err := yaml.Marshal(&Conf{
			Id:              8000,
			Addr:            "0.0.0.0:8000",
			InitConnTimeout: time.Second * 6,
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
}

type name struct {
	*node.Server
	common.Router
}

func (n *name) GetConn(id uint16) (common.Conn, bool) {
	return n.Conns.GetConn(id)
}

func (n *name) GetConns() []common.Conn {
	return n.Conns.GetConns()
}

func main() {
	data, err := os.ReadFile(*confFile)
	if err != nil {
		log.Fatalln("read config err", err)
	}
	c := new(Conf)
	if err = yaml.Unmarshal(data, c); err != nil {
		log.Fatalln("Unmarshal conf err", err)
	}
	s := new(ServerHandle)
	srv, err := node.ListenTCP(c.Addr, c.Id)
	if err != nil {
		log.Fatalln(err)
	}
	defer srv.Close()
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
		s.NodeDiscoveryProtocol = protocol.NewNodeDiscoveryProtocol(&name{srv, srv.Router}, os.Stdout)
		go s.NodeDiscoveryProtocol.StartTimingQueryEnableProtoNode(context.Background(), c.NodeDiscoveryProtocol.QueryInterval)
	}
	go func() {
		time.Sleep(time.Second)
		sc := bufio.NewScanner(os.Stdin)
		fmt.Print(">>")
		for sc.Scan() && srv.State == node.StateType_Listen {
			if sc.Text() != "" {
				executeCmd, err := cmd.Group.ExecuteContext(context.WithValue(context.Background(), "server", srv), strings.Fields(sc.Text()))
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

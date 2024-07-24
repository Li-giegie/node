package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/protocol"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type ServerHandle struct {
	protocol.NodeDiscoveryProtocol
	protocol.ServerAuthProtocol
	protocol.ServerHelloProtocol
	id   uint16
	addr string
	node.Server
}

func (h *ServerHandle) Init(conn net.Conn) (remoteId uint16, err error) {
	return h.ServerAuthProtocol.Init(conn)
}

func (h *ServerHandle) Connection(conn common.Conn) {
	log.Println("connection", conn.RemoteId())
	h.NodeDiscoveryProtocol.Connection(conn)
}

func (h *ServerHandle) Handle(ctx common.Context) {
	log.Printf("server [%d] Handl datae %s\n", ctx.SrcId(), ctx.Data())
	ctx.Reply([]byte(fmt.Sprintf("server [%d] handle reply: %s", h.id, ctx.Data())))
}

func (h *ServerHandle) ErrHandle(msg *common.Message, err error) {
	log.Println("ErrHandle", msg.String())
}

func (h *ServerHandle) CustomHandle(ctx common.Context) {
	if !h.NodeDiscoveryProtocol.CustomHandle(ctx) {
		return
	}
	if !h.ServerHelloProtocol.CustomHandle(ctx) {
		return
	}
	log.Println("CustomHandle", ctx.String())
}

func (h *ServerHandle) Disconnect(id uint16, err error) {
	log.Println("Disconnect", id, err)
	h.NodeDiscoveryProtocol.Disconnect(id, err)
}

var helpText = "invalid command\nrequest [-t: timeout/ms] [-id: destId] text\nrequest -t 3000 -id 1 hello\nsend [destId] text\nsend -id 1 hello"

func (h *ServerHandle) Listen() error {
	srv, err := node.ListenTCP(h.id, h.addr, h)
	if err != nil {
		return err
	}
	h.Server = srv
	h.NodeDiscoveryProtocol = protocol.NewNodeDiscoveryProtocol(srv, os.Stdout)
	h.ServerHelloProtocol = protocol.NewServerHelloProtocol(time.Second*5, time.Second*5, time.Second*20, os.Stdout)
	go h.ServerHelloProtocol.StartServer(srv)
	go h.NodeDiscoveryProtocol.StartTimingQueryEnableProtoNode(context.Background(), time.Second*10)
	go func() {
		set := flag.NewFlagSet("", flag.ContinueOnError)
		set.Usage = func() {
			fmt.Println(helpText)
			set.PrintDefaults()
		}
		dstId := new(uint)
		timeout := new(uint)
		set.UintVar(dstId, "id", 0, "dest id")
		set.UintVar(timeout, "t", 3000, "timeout")
		scan := bufio.NewScanner(os.Stdin)
		time.Sleep(time.Millisecond * 100)
		fmt.Print(">>")
		for scan.Scan() {
			cmds := strings.Split(scan.Text(), " ")
			if cmds[0] == "send" || cmds[0] == "request" {
				if err = set.Parse(cmds[1:]); err != nil {
					fmt.Printf("%s\n>>", helpText)
					continue
				}
			}
			data := []byte(strings.Join(set.Args(), " "))
			switch cmds[0] {
			case "":
			case "conns":
				for _, conn := range srv.GetConns() {
					fmt.Print(conn.RemoteId(), " ")
				}
				fmt.Println()
			case "route":
				srv.PrintString()
			case "request":
				ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(*timeout))
				replt, err := srv.Request(ctx, uint16(*dstId), data)
				if err != nil {
					fmt.Println("request err", err)
				} else {
					log.Println(string(replt))
				}
				cancel()
			case "send":
				if _, err = srv.WriteTo(uint16(*dstId), data); err != nil {
					fmt.Println("send err", err)
				}
			case "exit":
				_ = srv.Close()
			default:
				fmt.Println(helpText)
			}
			fmt.Print(">>")
		}
	}()
	return nil
}

func (h *ServerHandle) Serve() (err error) {
	defer func() {
		h.Server.Close()
		log.Printf("server [%d] exit err=%v\n", h.id, err)
	}()
	return h.Server.Serve()
}

func NewServerHandle(id uint16, addr string, key string, timeout time.Duration) *ServerHandle {
	return &ServerHandle{
		ServerAuthProtocol: protocol.NewServerAuthProtocol(id, key, timeout),
		id:                 id,
		addr:               addr,
	}
}

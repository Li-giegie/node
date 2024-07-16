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
	"time"
)

type ServerHandle struct {
	protocol.ServerAuthProtocol
	protocol.NodeDiscoveryProtocol
	id   uint16
	addr string
}

func (h *ServerHandle) Init(conn net.Conn) (remoteId uint16, err error) {
	return h.ServerAuthProtocol.Init(conn)
}

func (h *ServerHandle) Connection(conn common.Conn) {
	log.Println("connection", conn.RemoteId())
	h.NodeDiscoveryProtocol.Connection(conn)
}

func (h *ServerHandle) Handle(ctx common.Context) {
	log.Println("Handle", ctx.String())
	ctx.Reply([]byte(fmt.Sprintf("server [%d] handle reply: %s", h.id, ctx.Data())))
}

func (h *ServerHandle) ErrHandle(msg *common.Message, err error) {
	log.Println("ErrHandle", msg.String(), err)
}

func (h *ServerHandle) CustomHandle(ctx common.Context) {
	if !h.NodeDiscoveryProtocol.CustomHandle(ctx) {
		return
	}
	log.Println("CustomHandle", ctx.String())
}

func (h *ServerHandle) Disconnect(id uint16, err error) {
	log.Println("Disconnect", id, err)
	h.NodeDiscoveryProtocol.Disconnect(id, err)
}

func (h *ServerHandle) Serve() error {
	srv, err := node.ListenTCP(h.id, h.addr, h)
	if err != nil {
		return err
	}
	log.Printf("server [%d] start success\n", h.id)
	defer func() {
		srv.Close()
		log.Printf("server [%d] exit err=%v\n", h.id, err)
	}()
	h.NodeDiscoveryProtocol = protocol.NewNodeDiscoveryProtocol(srv)
	go h.NodeDiscoveryProtocol.StartTimingQueryEnableProtoNode(context.Background(), time.Second*10)
	go func() {
		scan := bufio.NewScanner(os.Stdin)
	loop1:
		fmt.Print(">>")
		for scan.Scan() {
			switch scan.Text() {
			case "":
				fmt.Print(">>")
			case "route":
				fmt.Print("route>>")
				for scan.Scan() {
					switch scan.Text() {
					case "":
						fmt.Print("route>>")
					case "list":
						srv.PrintString()
						fmt.Print("route>>")
					case "q", "quit", "exit":
						goto loop1
					default:
						fmt.Printf("未知命令 %s\nroute>>", scan.Text())
					}
				}
			default:
				fmt.Printf("未知命令 %s\n>>", scan.Text())
			}

		}
	}()
	return srv.Serve()
}
func NewHandle(id uint16, addr string, key string, timeout time.Duration) *ServerHandle {
	return &ServerHandle{
		ServerAuthProtocol: protocol.NewServerAuthProtocol(id, key, timeout),
		id:                 id,
		addr:               addr,
	}
}

var id = flag.Uint("id", 0, "id")
var laddr = flag.String("laddr", "0.0.0.0:8000", "local address")
var key = flag.String("key", "hello", "auth key")

func main() {
	flag.Parse()
	h := NewHandle(uint16(*id), *laddr, *key, time.Second*5)
	if err := h.Serve(); err != nil {
		log.Println(err)
	}
}

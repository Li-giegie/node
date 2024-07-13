package main

import (
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/protocol"
	"log"
	"net"
	"time"
)

type ServerHandle struct {
	*protocol.NodeDiscoveryProtocol
	*protocol.AuthProtocol
	id      uint16
	key     string
	addr    string
	timeout time.Duration
}

func (h *ServerHandle) Init(conn net.Conn) (remoteId uint16, err error) {
	return h.ConnectionServer(conn, h.id, h.key, h.timeout)
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
	h.NodeDiscoveryProtocol = protocol.NewNodeDiscoveryProtocol()
	go h.NodeDiscoveryProtocol.InitServer(srv, time.Second*3)
	return srv.Serve()
}
func NewHandle(id uint16, addr string, key string, timeout time.Duration) *ServerHandle {
	return &ServerHandle{
		AuthProtocol: new(protocol.AuthProtocol),
		id:           id,
		key:          key,
		addr:         addr,
		timeout:      timeout,
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

package main

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/protocol"
	"log"
	"net"
)

type ServerHandle struct {
	protocol.NodeDiscoveryProtocol
	protocol.ServerAuthProtocol
	protocol.ServerHelloProtocol
	node.Server
}

func (h *ServerHandle) Init(conn net.Conn) (remoteId uint16, err error) {
	return h.ServerAuthProtocol.Init(conn)
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

func (h *ServerHandle) ErrHandle(msg *common.Message, err error) {
	log.Println("ErrHandle", msg.String())
}

func (h *ServerHandle) CustomHandle(ctx common.Context) {
	if h.ServerHelloProtocol != nil {
		if !h.ServerHelloProtocol.CustomHandle(ctx) {
			return
		}
	}
	if h.NodeDiscoveryProtocol != nil {
		if !h.NodeDiscoveryProtocol.CustomHandle(ctx) {
			return
		}
	}
	log.Println("CustomHandle", ctx.String())
}

func (h *ServerHandle) Disconnect(id uint16, err error) {
	log.Println("Disconnect", id, err)
	if h.NodeDiscoveryProtocol != nil {
		h.NodeDiscoveryProtocol.Disconnect(id, err)
	}
}

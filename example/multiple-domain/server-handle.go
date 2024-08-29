package main

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/protocol"
	"log"
)

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

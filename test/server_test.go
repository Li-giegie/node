package test

import (
	"context"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"log"
	"net"
	"testing"
)

type Handler struct {
	node.Server
}

func (h Handler) Connection(conn net.Conn) (remoteId uint16, err error) {
	log.Println("Connection ", conn.RemoteAddr().String())
	return 1, nil
}

func (h Handler) Handle(ctx *common.Context) {
	log.Println("Handle ", ctx.String())
	go func() {
		conn, ok := h.GetConn(1)
		if !ok {
			ctx.Reply([]byte("conn not exist"))
			return
		}
		data, err := conn.Request(context.Background(), []byte("server: hello 1"))
		if err != nil {
			ctx.Reply([]byte("write err"))
			return
		}
		ctx.Reply(data)
	}()
}

func (h Handler) ErrHandle(msg *common.Message) {
	log.Println("ErrHandle ", msg.String())
}

func (h Handler) DropHandle(msg *common.Message) {
	log.Println("DropHandle ", msg.String())
}

func (h Handler) CustomHandle(ctx *common.Context) {
	log.Println("CustomHandle ", ctx.String())
}

func (h Handler) Disconnect(id uint16, err error) {
	log.Println("Disconnect ", id, err)
}

func TestServer(t *testing.T) {
	l, err := node.ListenTCP(0, "0.0.0.0:8080")
	if err != nil {
		t.Error(err)
		return
	}
	if err = l.Serve(&Handler{Server: l}); err != nil {
		t.Error(err)
		return
	}
}

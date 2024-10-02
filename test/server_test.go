package test

import (
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"log"
	"testing"
	"time"
)

type Handler struct {
	*node.Server
}

func (h Handler) Connection(conn common.Conn) {
	log.Println("connection", conn.RemoteId())
}

func (h Handler) Handle(ctx common.Context) {
	ctx.Reply([]byte("pong"))
}

func (h Handler) ErrHandle(ctx common.ErrContext, err error) {
	log.Println("ErrHandle", err, ctx.String())
}

func (h Handler) CustomHandle(ctx common.CustomContext) {
	log.Println("CustomHandle", ctx.String())
}

func (h Handler) Disconnect(id uint16, err error) {
	log.Println("Disconnect", id, err)
}

func TestServer(t *testing.T) {
	srv, err := node.ListenTCP("0.0.0.0:8000", &node.Identity{
		Id:            8000,
		AccessKey:     []byte("hello"),
		AccessTimeout: time.Second * 6,
	})
	if err != nil {
		t.Error(err)
		return
	}
	defer srv.Close()
	if err = srv.Serve(&Handler{srv}); err != nil {
		t.Error(err)
		return
	}
}

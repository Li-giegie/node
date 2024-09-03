package test

import (
	"errors"
	"fmt"
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
	var err error
	switch ctx.Id() % 5 {
	case 0:
		err = ctx.Reply(ctx.Data())
	case 1:
		err = ctx.ErrReply(ctx.Data(), errors.New("test error"))
	case 2:
		err = ctx.ErrReply(ctx.Data(), errors.New(string(make([]byte, 65535))))
	case 3:
		err = ctx.Reply(ctx.Data())
		if err != nil {
			fmt.Println("case 3 reply err", err)
			return
		}
		err = ctx.ErrReply(ctx.Data(), errors.New("test error"))
	case 4:
		err = ctx.ErrReply(ctx.Data(), errors.New(""))
	}
	if err != nil {
		fmt.Println("reply err", err)
	}
}

func (h Handler) ErrHandle(ctx common.ErrContext, err error) {
	log.Println("ErrHandle", err, ctx.String())
}

func (h Handler) CustomHandle(ctx common.CustomContext) {
	log.Println("CustomHandle", ctx.String())
	ctx.CustomReply(ctx.Type(), ctx.Data())
	//ctx.RecycleMsg()
}

func (h Handler) Disconnect(id uint16, err error) {
	log.Println("Disconnect", id, err)
	//h.Server.Close()
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

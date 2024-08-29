package test

import (
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"log"
	"testing"
)

type Handler struct {
	*node.Server
}

func (h Handler) Connection(conn common.Conn) {
	log.Println("connection", conn.RemoteId())
}

func (h Handler) Handle(ctx common.Context) {
	//log.Println("Handle", ctx.String())
	ctx.Reply(ctx.Data())
	ctx.RecycleMsg()
	//if err := ctx.Reply(append([]byte("server reply"), ctx.Data()...)); err != nil {
	//	fmt.Println("err", err)
	//}
}

func (h Handler) ErrHandle(ctx common.ErrContext, err error) {

	log.Println("ErrHandle", err, ctx.String())
}

func (h Handler) CustomHandle(ctx common.CustomContext) {
	log.Println("CustomHandle", ctx.String())
	ctx.CustomReply(ctx.Type(), ctx.Data())
}

func (h Handler) Disconnect(id uint16, err error) {
	log.Println("Disconnect", id, err)
	//h.Server.Close()
}

func TestServer(t *testing.T) {
	srv, err := node.ListenTCP("0.0.0.0:8000", 8000)
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

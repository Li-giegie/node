package test

import (
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"log"
	"testing"
	"time"
)

type Echo struct {
}

func (e Echo) Connection(conn common.Conn) {

}

func (e Echo) Handle(ctx common.Context) {
	ctx.Reply(ctx.Data())
}

func (e Echo) ErrHandle(ctx common.ErrContext, err error) {
	log.Println("ErrHandle", ctx.String())
}

func (e Echo) CustomHandle(ctx common.CustomContext) {
	log.Println("CustomHandle", ctx.String())
}

func (e Echo) Disconnect(id uint16, err error) {
	log.Println("Disconnect", id)
}

func TestEchoServer(t *testing.T) {
	srv, err := node.ListenTCP("0.0.0.0:8888", &node.Identity{
		Id:            0,
		AccessKey:     []byte("echo"),
		AccessTimeout: time.Second * 3,
	})
	if err != nil {
		t.Error(err)
		return
	}
	defer srv.Close()
	if err = srv.Serve(&Echo{}); err != nil {
		log.Println(err)
	}
}

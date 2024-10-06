package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"log"
	"testing"
	"time"
)

type CliHandler struct{}

func (h CliHandler) Connection(conn common.Conn) {
	log.Println("Handle", conn.RemoteId())
}

func (h CliHandler) Handle(ctx common.Context) {
	log.Println("Handle", ctx.String())
}

func (h CliHandler) ErrHandle(ctx common.ErrContext, err error) {
	log.Println("ErrHandle", err, ctx.String())
}

func (h CliHandler) CustomHandle(ctx common.CustomContext) {
	log.Println("CustomHandle", ctx.String())
}

func (h CliHandler) Disconnect(id uint32, err error) {
	fmt.Println("Disconnect", id, err)
}

func TestClient(t *testing.T) {
	conn, err := node.DialTCP(
		"0.0.0.0:8000",
		&node.Identity{
			Id:            8001,
			AccessKey:     []byte("hello"),
			AccessTimeout: time.Second * 6,
		},
		&CliHandler{},
	)
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	res, err := conn.Request(ctx, []byte("ping"))
	if err != nil {
		t.Error(err)
		return
	}
	println(string(res))
}

package test

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	l, err := node.ListenTCP(1, "127.0.0.1:8080")
	if err != nil {
		t.Error(err)
		return
	}
	if err = l.Tick(time.Second, time.Second*1, time.Second*4, true); err != nil {
		t.Error(err)
		return
	}
	l.HandleFunc(1, func(ctx *common.Context) {
		ctx.Write([]byte("ok"))
	})
	l.HandleFunc(2, func(ctx *common.Context) {
		fmt.Println("msg", ctx.String())
		ctx.Write([]byte("ok"))
	})
	if err = l.Serve(); err != nil {
		t.Error(err)
		return
	}
}

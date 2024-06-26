package test

import (
	"context"
	"errors"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/utils"
	"log"
	"net"
	"testing"
	"time"
)

type ClientHandle struct {
	common.Conn
	id    uint16
	key   string
	addr  string
	stopC chan error
}

type Auth struct {
	ClientId uint16
	Key      string
	ServerId uint16
	Msg      string
	Permit   bool
}

func (c *ClientHandle) Connection(conn net.Conn) (remoteId uint16, err error) {
	auth := &Auth{ClientId: c.id, Key: c.key}
	if err = utils.JSONPackEncode(conn, auth); err != nil {
		log.Println("auth err 1", err)
		return 0, nil
	}
	if err = utils.JSONPackDecode(time.Second*6, conn, auth); err != nil {
		log.Println("auth err 2", err)
		return 0, err
	}
	if !auth.Permit {
		return 0, errors.New(auth.Msg)
	}
	log.Println("Connection ", conn.RemoteAddr().String())
	return auth.ServerId, nil
}

func (c *ClientHandle) Handle(ctx common.Context) {
	go func() {
		log.Println("Handle ", ctx.String())
		if err := ctx.Reply([]byte("client 1 handle success")); err != nil {
			fmt.Println(err)
		}
	}()
}

func (c *ClientHandle) ErrHandle(msg *common.Message) {
	log.Println("ErrHandle ", msg.String())
}

func (c *ClientHandle) DropHandle(msg *common.Message) {
	log.Println("DropHandle ", msg.String())
}

func (c *ClientHandle) CustomHandle(ctx common.Context) {
	log.Println("CustomHandle ", ctx.String())
}

func (c *ClientHandle) Disconnect(id uint16, err error) {
	log.Println("Disconnect ", id, err)
	c.stopC <- err
}

func (c *ClientHandle) Serve() error {
	conn, err := node.Dial("tcp", c.addr, c.id, c)
	if err != nil {
		return err
	}
	c.Conn = conn
	return nil
}

func TestClient(t *testing.T) {
	c := &ClientHandle{
		id:    2,
		key:   "hello",
		addr:  "0.0.0.0:8080",
		stopC: make(chan error, 1),
	}
	err := c.Serve()
	if err != nil {
		t.Error(err)
		return
	}
	resp, err := c.Request(context.Background(), []byte("request server"))
	fmt.Printf("%s %v\n", resp, err)
	resp, err = c.Request(context.Background(), nil)
	fmt.Printf("%s %q\n", resp, err)
	resp, err = c.Request(context.Background(), make([]byte, 1))
	fmt.Printf("%s %v\n", resp, err)
	c.Close()
	<-c.stopC
}

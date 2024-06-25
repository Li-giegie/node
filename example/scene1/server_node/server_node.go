package main

import (
	"errors"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/utils"
	"log"
	"net"
	"time"
)

type Handler struct {
	node.Server
}
type Auth struct {
	ClientId uint16
	Key      string
	ServerId uint16
	Msg      string
	Permit   bool
}

func (h Handler) Connection(conn net.Conn) (remoteId uint16, err error) {
	auth := new(Auth)
	defer func() {
		wErr := utils.JSONPackEncode(conn, auth)
		if wErr != nil {
			err = wErr
			conn.Close()
			return
		}
		if err != nil {
			_ = conn.Close()
		}
	}()
	if err = utils.JSONPackDecode(time.Second*6, conn, auth); err != nil {
		return 0, err
	}
	if auth.Key != "hello" {
		auth.Msg = "key invalid"
		auth.Permit = false
		return 0, errors.New("key invalid")
	}
	auth.Msg = "auth success"
	auth.Permit = true
	auth.ServerId = h.Id()
	return auth.ClientId, nil
}

func (h Handler) Handle(ctx *common.Context) {
	//TODO implement me
	panic("implement me")
}

func (h Handler) ErrHandle(msg *common.Message) {
	//TODO implement me
	panic("implement me")
}

func (h Handler) DropHandle(msg *common.Message) {
	//TODO implement me
	panic("implement me")
}

func (h Handler) CustomHandle(ctx *common.Context) {
	//TODO implement me
	panic("implement me")
}

func (h Handler) Disconnect(id uint16, err error) {
	//TODO implement me
	panic("implement me")
}

func main() {
	srv, err := node.ListenTCP(0, "0.0.0.0:8080")
	if err != nil {
		log.Fatalln(err)
	}
	defer srv.Close()
	err = srv.Serve(&Handler{srv})
	if err != nil {
		log.Println(err)
	}
}

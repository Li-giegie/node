package test

import (
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

type Handler struct {
	node.Server
}

func (h Handler) Connection(conn net.Conn) (remoteId uint16, err error) {
	auth := new(Auth)
	defer func() {
		wErr := utils.JSONPackEncode(conn, auth)
		if wErr != nil {
			err = wErr
			conn.Close()
			log.Println("认证失败1 ", err)
			return
		}
		if err != nil {
			_ = conn.Close()
			log.Println("认证失败2 ", err)
			return
		}
		log.Println("认证通过", remoteId)
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
func (h Handler) Handle(ctx common.Context) {
	log.Println("Handle ", ctx.String())
	if len(ctx.Data()) == 0 {
		ctx.ErrReply([]byte("err"), errors.New("data len 0"))
		return
	}
	var data []byte
	fmt.Println(ctx.Data())
	switch ctx.Data()[0] {
	case 0:
		data = []byte("1")
	default:
		data = []byte("default")
	}
	ctx.Reply(data)
}

func (h Handler) ErrHandle(msg *common.Message) {
	log.Println("ErrHandle ", msg.String())
}

func (h Handler) DropHandle(msg *common.Message) {
	log.Println("DropHandle ", msg.String())
}

func (h Handler) CustomHandle(ctx common.Context) {
	log.Println("CustomHandle ", ctx.String())
	ctx.CustomReply(ctx.Type(), ctx.Data())
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

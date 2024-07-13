package test

import (
	"errors"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/protocol"
	"log"
	"net"
	"testing"
	"time"
)

type Handler struct {
	node.Server
	authKey     string
	authTimeout time.Duration
	*protocol.AuthProtocol
	*protocol.HelloProtocol
}

func (h *Handler) Init(conn net.Conn) (remoteId uint16, err error) {
	return h.AuthProtocol.ConnectionServer(conn, h.Id(), h.authKey, h.authTimeout)
}

func (h *Handler) Connection(conn common.Conn) {

}

func (h *Handler) Handle(ctx common.Context) {
	log.Println("Handle ", ctx.String())
	switch len(ctx.Data()) {
	case 0:
		ctx.ErrReply(nil, nil)
	case 1:
		ctx.ErrReply(nil, errors.New(""))
	case 2:
		ctx.ErrReply(nil, errors.New("error test 2"))
	case 3:
		ctx.ErrReply([]byte("123"), errors.New("error test 3"))
	case 4:
		ctx.ErrReply([]byte("1234"), nil)
	case 5:
		fmt.Println(ctx.ErrReply(make([]byte, 65533), errors.New(string(make([]byte, 65533)))))
	case 6:
		fmt.Println(ctx.ErrReply(make([]byte, 65535), errors.New(string(make([]byte, 65535)))))
	default:
		ctx.Reply(ctx.Data())
	}

}

func (h *Handler) ErrHandle(msg *common.Message, err error) {
	log.Println("ErrHandle ", msg.String())
}

func (h *Handler) CustomHandle(ctx common.Context) {
	if h.HelloProtocol.CustomHandle(ctx) {
		log.Println("CustomHandle ", ctx.String())
	}
}

func (h *Handler) Disconnect(id uint16, err error) {
	log.Println("Disconnect ", id, err)
}

func (h *Handler) Serve() error {
	l, err := node.ListenTCP(0, "0.0.0.0:8080", h)
	if err != nil {
		return err
	}
	h.authKey = "hello"
	h.authTimeout = time.Second * 6
	h.Server = l
	h.AuthProtocol = new(protocol.AuthProtocol)
	h.HelloProtocol = new(protocol.HelloProtocol)
	go h.HelloProtocol.InitServer(l, time.Second, time.Second*3, time.Second*15, &LogWriter{})
	if err = l.Serve(); err != nil {
		return err
	}
	return nil
}

func TestServer(t *testing.T) {
	h := Handler{}
	err := h.Serve()
	if err != nil {
		t.Error(err)
	}
}

type LogWriter struct {
}

func (l *LogWriter) Write(b []byte) (n int, err error) {
	log.Print(string(b))
	return
}

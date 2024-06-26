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

type Server struct {
	node.Server
	key  string
	id   uint16
	addr string
}

type Auth struct {
	ClientId uint16 `json:"client_id,omitempty"`
	ServerId uint16 `json:"server_id,omitempty"`
	Msg      string `json:"msg,omitempty"`
	Permit   bool   `json:"permit,omitempty"`
}

// Connection 建立连接回调，再该回调中作认证
func (s *Server) Connection(conn net.Conn) (remoteId uint16, err error) {
	log.Println("client connection")
	// 创建认证结构
	auth := new(Auth)
	// 认证成功失败通知客户端
	defer func() {
		// 认证失败，告知失败原因及Permit字段置为false
		if err != nil {
			auth.Permit = false
			auth.Msg = err.Error()
		} else { //反之
			auth.ServerId = s.Id()
			auth.Permit = true
			auth.Msg = "success"
		}
		// 返回结果
		if err2 := utils.JSONPackEncode(conn, auth); err2 != nil {
			err = err2
		}
	}()

	// 提供json解码客户端发送的数据
	if err = utils.JSONPackDecode(time.Second*6, conn, auth); err != nil {
		return 0, err
	}
	// 比较客户端传来的msg和key比较，只有相等才会通过
	if auth.Msg != s.key {
		return 0, errors.New("invalid key")
	}
	return auth.ClientId, nil
}

func (s *Server) Handle(ctx common.Context) {
	log.Println("handle: ", ctx.String())
	if len(ctx.Data()) == 0 {
		ctx.ErrReply(nil, errors.New("invalid data"))
		return
	}
	ctx.Reply([]byte("server Handle: ok"))
}

func (s *Server) ErrHandle(msg *common.Message) {
	log.Println("ErrHandle: ", msg.String())
}

func (s *Server) DropHandle(msg *common.Message) {
	log.Println("DropHandle: ", msg.String())
}

func (s *Server) CustomHandle(ctx common.Context) {
	log.Println("CustomHandle: ", ctx.String())
	ctx.CustomReply(ctx.Type(), []byte("server CustomHandle: ok"))
}

func (s *Server) Disconnect(id uint16, err error) {
	log.Println("Disconnect: ", id, err)
}

func (s *Server) Serve() error {
	l, err := node.ListenTCP(s.id, s.addr)
	if err != nil {
		return err
	}
	defer l.Close()
	s.Server = l
	return l.Serve(s)
}

func main() {
	srv := new(Server)
	srv.id = 0
	srv.key = "hello"
	srv.addr = "0.0.0.0:8080"
	if err := srv.Serve(); err != nil {
		log.Fatalln(err)
	}
}

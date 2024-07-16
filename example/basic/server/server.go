package main

import (
	"errors"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/protocol"
	"log"
	"net"
	"os"
	"time"
)

type Server struct {
	protocol.ServerAuthProtocol
	protocol.ServerHelloProtocol
	node.Server
	key         string
	localId     uint16
	addr        string
	authTimeout time.Duration
}

// Connection 建立连接回调，再该回调中作认证
func (s *Server) Init(conn net.Conn) (remoteId uint16, err error) {
	return s.ServerAuthProtocol.Init(conn)
}

func (s *Server) Connection(conn common.Conn) {
	log.Println("connection", conn.RemoteId())
}

func (s *Server) Handle(ctx common.Context) {
	log.Println("handle: ", ctx.String())
	if len(ctx.Data()) == 0 {
		ctx.ErrReply(nil, errors.New("invalid data"))
		return
	}
	ctx.Reply([]byte("server_node_0 Handle: ok"))
}

func (s *Server) ErrHandle(msg *common.Message, err error) {
	log.Println("ErrHandle: ", msg.String())
}

func (s *Server) CustomHandle(ctx common.Context) {
	if s.ServerHelloProtocol.CustomHandle(ctx) {
		log.Println("CustomHandle: ", ctx.String())
		ctx.CustomReply(ctx.Type(), []byte("server_node_0 CustomHandle: ok"))
	}
}

func (s *Server) Disconnect(id uint16, err error) {
	log.Println("Disconnect: ", id, err)
}

func (s *Server) Serve() error {
	l, err := node.ListenTCP(s.localId, s.addr, s)
	if err != nil {
		return err
	}
	defer l.Close()
	s.Server = l
	go s.ServerHelloProtocol.StartServer(l)
	return l.Serve()
}

func main() {
	srv := new(Server)
	srv.localId = 0
	srv.authTimeout = time.Second * 6
	srv.key = "hello"
	srv.addr = "0.0.0.0:8080"
	srv.ServerAuthProtocol = protocol.NewServerAuthProtocol(srv.localId, srv.key, srv.authTimeout)
	srv.ServerHelloProtocol = protocol.NewServerHelloProtocol(time.Second*3, time.Second*3, time.Second*60, os.Stdout)
	if err := srv.Serve(); err != nil {
		log.Fatalln(err)
	}
}

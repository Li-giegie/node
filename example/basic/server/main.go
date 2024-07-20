package main

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/example/basic"
	"github.com/Li-giegie/node/protocol"
	"log"
	"net"
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

func (s *Server) Init(conn net.Conn) (remoteId uint16, err error) {
	return s.ServerAuthProtocol.Init(conn)
}

func (s *Server) Connection(conn common.Conn) {
	log.Println("connection", conn.RemoteId())
}

func (s *Server) Handle(ctx common.Context) {
	log.Println("handle: ", ctx.String())
	ctx.Reply([]byte(fmt.Sprintf("server [%d] handle reply: %s", s.localId, ctx.Data())))
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
	log.Printf("server [%d] start success\n", s.localId)
	s.Server = l
	go s.ServerHelloProtocol.StartServer(l)
	return l.Serve()
}

func main() {
	startFSet := basic.NewStartFlagSet().Parse()
	srv := new(Server)
	srv.localId = uint16(startFSet.LId)
	srv.authTimeout = time.Millisecond * time.Duration(startFSet.AuthTimeout)
	srv.key = startFSet.Key
	srv.addr = startFSet.LAddr
	srv.ServerAuthProtocol = protocol.NewServerAuthProtocol(srv.localId, srv.key, srv.authTimeout)
	srv.ServerHelloProtocol = protocol.NewServerHelloProtocol(time.Second*3, time.Second*10, time.Second*30, nil)
	go func() {
		if err := srv.Serve(); err != nil {
			log.Fatalln(err)
		}
	}()
	time.Sleep(time.Second)
	basic.ParseCmd(nil, srv)
}

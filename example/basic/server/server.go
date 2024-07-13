package main

import (
	"errors"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/protocol"
	"log"
	"net"
	"time"
)

type Server struct {
	*protocol.AuthProtocol
	*protocol.HelloProtocol
	node.Server
	key         string
	localId     uint16
	addr        string
	authTimeout time.Duration
}

// Connection 建立连接回调，再该回调中作认证
func (s *Server) Init(conn net.Conn) (remoteId uint16, err error) {
	return s.AuthProtocol.ConnectionServer(conn, s.localId, s.key, s.authTimeout)
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
	if s.HelloProtocol.CustomHandle(ctx) {
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
	s.AuthProtocol = new(protocol.AuthProtocol)
	s.HelloProtocol = new(protocol.HelloProtocol)
	go s.HelloProtocol.InitServer(s, time.Second, time.Second*5, time.Second*25, &LogWriter{})
	return l.Serve()
}

func main() {
	srv := new(Server)
	srv.localId = 0
	srv.authTimeout = time.Second * 6
	srv.key = "hello"
	srv.addr = "0.0.0.0:8080"
	if err := srv.Serve(); err != nil {
		log.Fatalln(err)
	}
}

type LogWriter struct {
}

func (l *LogWriter) Write(b []byte) (n int, err error) {
	log.Print(string(b))
	return
}

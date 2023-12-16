package node

import (
	"context"
	"log"
	"net"
	"time"
)

type ServerI interface {
	HandleFunc(api uint32, handle HandleFunc) *Handler
	HandlerI(ri ...HandlerI) *Handler
	ListenAndServer() error
	Request(ctx context.Context, dstId uint64, api uint32, data []byte) ([]byte, error)
	Send(dstId uint64, api uint32, data []byte) error
	CloseConn(id uint64)
	ConnList() []Conn
	FindConn(id uint64) (Conn, bool)
	Shutdown()
}

type Server struct {
	id          uint64
	addr        string
	state       bool
	maxConNum   int
	listen      *net.TCPListener
	srvConnMgmt *serverConnectionManager
}

// NewServer 创建一个node服务端
func NewServer(address string, options ...Option) ServerI {
	var srv = new(Server)
	srv.addr = address
	srv.id = DEFAULT_ServerID
	srv.maxConNum = DEFAULT_MAXCONNNUM
	srv.srvConnMgmt = newServerConnectionManager(srv)
	srv.state = true
	for _, v := range options {
		v(srv)
	}
	return srv
}

type Option func(srv *Server) *Server

func WithSrvId(id uint64) Option {
	return func(srv *Server) *Server {
		srv.id = id
		return srv
	}
}

func WithSrvConnTimeout(t time.Duration) Option {
	return func(srv *Server) *Server {
		srv.srvConnMgmt.connTimeOut = t
		return srv
	}
}

func WithSrvGoroutine(min, max int) Option {
	return func(srv *Server) *Server {
		srv.srvConnMgmt.maxGoroutine = max
		srv.srvConnMgmt.minGoroutine = min
		return srv
	}
}

// WithSrvAuthentication Set authentication
func WithSrvAuthentication(authFunc AuthenticationFunc) Option {
	return func(srv *Server) *Server {
		srv.srvConnMgmt.AuthenticationFunc = authFunc
		return srv
	}
}

// WithSrvConnectionEnableFunc callback function
func WithSrvConnectionEnableFunc(successFunc ConnectionEnableFunc) Option {
	return func(srv *Server) *Server {
		srv.srvConnMgmt.ConnectionEnableFunc = successFunc
		return srv
	}
}

// WithSrvMaxConnectNum <= 0 disable The number of connections is not limited
func WithSrvMaxConnectNum(maxNum int) Option {
	return func(srv *Server) *Server {
		srv.maxConNum = maxNum
		return srv
	}
}

func (s *Server) getId() uint64 {
	return s.id
}

func (s *Server) HandleFunc(api uint32, handle HandleFunc) *Handler {
	return s.srvConnMgmt.HandleFunc(api, handle)
}

func (s *Server) HandlerI(ri ...HandlerI) *Handler {
	return s.srvConnMgmt.HandlerI(ri...)
}

// ListenAndServer 开启服务
func (s *Server) ListenAndServer() error {
	err := s.srvConnMgmt.init()
	if err != nil {
		return err
	}
	addr, err := parseAddress("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listen, err = net.ListenTCP("tcp", addr[0])
	if err != nil {
		return err
	}
	defer s.listen.Close()
	log.Printf("server start success id：%v listen：%v\n", s.id, addr[0].String())
	for s.state {
		if len(s.srvConnMgmt.connList.GetMap()) > s.maxConNum {
			time.Sleep(time.Millisecond * 200)
			continue
		}
		conn, err := s.listen.AcceptTCP()
		if err != nil {
			if !s.state {
				log.Println("server shutdown ------")
				return nil
			}
			return err
		}
		go s.srvConnMgmt.addConnect(conn)
	}
	return nil
}

func (s *Server) Request(ctx context.Context, dstId uint64, api uint32, data []byte) ([]byte, error) {
	conn, ok := s.FindConn(dstId)
	if !ok {
		return nil, ErrConnNotExist
	}
	return conn.Request(ctx, api, data)
}

func (s *Server) Send(dstId uint64, api uint32, data []byte) error {
	conn, ok := s.FindConn(dstId)
	if !ok {
		return ErrConnNotExist
	}
	return conn.Send(api, data)
}

func (s *Server) CloseConn(id uint64) {
	s.srvConnMgmt.CloseConn(id)
}

func (s *Server) ConnList() []Conn {
	return s.srvConnMgmt.ConnList()
}

func (s *Server) FindConn(id uint64) (Conn, bool) {
	return s.srvConnMgmt.FindConn(id)
}

func (s *Server) Shutdown() {
	s.state = false
	s.srvConnMgmt.CloseAllConn()
	_ = s.listen.Close()
}

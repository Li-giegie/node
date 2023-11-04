package node

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"net"
	"sync"
	"time"
)

type AuthenticationFunc func(id uint64, data []byte) (ok bool, reply []byte)

type Server struct {
	id            uint64
	addr          string
	maxGoroutine  int
	minGoroutine  int
	maxConnectNum int
	connTimeout   time.Duration
	state         bool
	connList      *ServerConnectList
	l             sync.RWMutex
	AuthenticationFunc
	*Handler
	listen *net.TCPListener
}

func NewServer(address string, options ...Option) (*Server, error) {
	var srv = new(Server)
	srv.addr = address
	srv.id = DEFAULT_ServerID
	srv.maxGoroutine = DEFAULT_MAX_GOROUTINE
	srv.minGoroutine = DEFAULT_MIN_GOROUTINE
	srv.maxConnectNum = DEFAULT_MAXCONNNUM
	srv.connTimeout = DEFAULT_KeepAlive
	srv.state = true
	for _, v := range options {
		v.(func(srv *Server) *Server)(srv)
	}
	var err error
	srv.Handler = newRouter()
	srv.connList, err = newServerConnectManager(srv.connTimeout, srv.Handler, srv.maxConnectNum, srv.minGoroutine)
	if err != nil {
		return srv, err
	}
	return srv, nil
}

type Option interface{}

func WithSrvId(id uint64) Option {
	return func(srv *Server) *Server {
		srv.id = id
		return srv
	}
}

func WithSrvConnTimeout(t time.Duration) Option {
	return func(srv *Server) *Server {
		srv.connTimeout = t
		return srv
	}
}

func WithSrvGoroutine(min, max int) Option {
	return func(srv *Server) *Server {
		srv.maxGoroutine = max
		srv.minGoroutine = min
		return srv
	}
}

// WithSrvMaxConnectNum <= 0 disable The number of connections is not limited
func WithSrvMaxConnectNum(maxNum int) Option {
	return func(srv *Server) *Server {
		srv.maxConnectNum = maxNum
		return srv
	}
}

// ListenAndServer 开启服务
func (s *Server) ListenAndServer() error {
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
		conn, err := s.listen.AcceptTCP()
		if !s.state {
			log.Println("server shutdown ------")
			return nil
		}
		if err != nil {
			fmt.Println("exit listen")
			return err
		}
		s.initializeConnection(conn)
	}
	return nil
}

// 初始化一个连接
func (s *Server) initializeConnection(conn *net.TCPConn) {
	go func() {
		id, err := s.authentication(conn)
		if err != nil {
			log.Printf("initializeConnection id[%v] err : -1 %v\n", id, err)
			_ = conn.Close()
			return
		}
		if err = s.connList.addConnect(id, conn); err != nil {
			log.Printf("initializeConnection id[%v] err : -2 %v\n", id, err)
			_ = conn.Close()
			return
		}
		log.Printf("initializeConnection id[%v] successfully\n", id)
	}()
}

// 认证现返回一个id和错误
func (s *Server) authentication(conn *net.TCPConn) (uint64, error) {
	nowNum := len(s.connList.connList)
	if s.maxConnectNum > 0 && nowNum >= s.maxConnectNum {
		_ = write(conn, []byte(auth_err_conn_supper_limit.Error()))
		return 0, auth_err_conn_supper_limit
	}
	buf, err := jeans.Unpack(conn)
	if err != nil {
		return 0, err
	}

	if len(buf) < 8 {
		_ = write(conn, []byte(auth_err_illegality.Error()))
		return 0, auth_err_illegality
	}
	id := binary.LittleEndian.Uint64(buf[:8])
	var data []byte
	if len(buf) > 8 {
		data = buf[8:]
	}

	s.l.RLock()
	mgConn, ok := s.connList.connList[id]
	s.l.RUnlock()
	if ok && mgConn.state {
		_ = write(conn, []byte(auth_err_user_online.Error()))
		return id, auth_err_user_online
	}

	if s.AuthenticationFunc == nil {
		err = write(conn, []byte(auth_sucess))
		return id, err
	}

	ok, b := s.AuthenticationFunc(id, data)
	if !ok {
		err = errors.New(auth_err_head + string(b))
		_ = write(conn, append([]byte(auth_err_head), b...))
		return id, err
	}

	err = write(conn, append([]byte(auth_sucess), b...))
	return id, err
}

func (s *Server) Request(ctx context.Context, id uint64, api uint32, data []byte) ([]byte, error) {
	conn, ok := s.connList.connList[id]
	if !ok {
		return nil, ErrConnNotExist
	}
	return conn.request(ctx, api, data)
}

func (s *Server) Send(id uint64, api uint32, data []byte) error {
	conn, ok := s.connList.connList[id]
	if !ok {
		return ErrConnNotExist
	}
	return conn.send(api, data)
}

func (s *Server) CloseConn(id uint64) {
	v, ok := s.connList.connList[id]
	if ok {
		v.Close()
	}
}

func (s *Server) OnLineConn() []uint64 {
	s.connList.lock.RLock()
	defer s.connList.lock.Unlock()
	conn := make([]uint64, 0, len(s.connList.connList))
	for k, _ := range s.connList.connList {
		conn = append(conn, k)
	}
	return conn
}

func (s *Server) Shutdown() {
	s.state = false
	s.connList.closeAllConn()
	_ = s.listen.Close()
}

package node

import (
	"bytes"
	"errors"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"net"
	"runtime"
	"time"
)

type AuthenticationFunc func(id string, data []byte) (ok bool, reply []byte)

type Server struct {
	Id                string
	addr              *net.TCPAddr
	WorkerProcessNum  int
	MaxConnectNum     int
	ConnectionTimeout time.Duration
	state             bool
	ctxChan           chan *Context
	scm               *ServerConnectManager
	AuthenticationFunc
	*RouteManager
	g *GoroutineManager
}

func NewServer(id, address string) *Server {
	var srv = new(Server)
	srv.Id = id
	srv.addr = mustAddress("tcp", address)[0]
	srv.WorkerProcessNum = runtime.NumCPU()
	srv.RouteManager = newRouter()
	srv.MaxConnectNum = DEFAULT_MAXCONNNUM
	srv.state = true
	srv.ConnectionTimeout = time.Second * 90
	return srv
}

// 初始化管理器
func (s *Server) initManager() {
	s.ctxChan = make(chan *Context, s.WorkerProcessNum*2)
	//连接管理器初始化
	s.scm = newServerConnectManager(s.ctxChan, s.ConnectionTimeout)
	//开启工作协程池
	s.g = newGoroutineManager(s.WorkerProcessNum, s.RouteManager, s.ctxChan, s.scm)
	s.g.start()
}

// ListenAndServer 开启服务
func (s *Server) ListenAndServer() error {
	s.initManager()
	listen, lErr := net.ListenTCP("tcp", s.addr)
	if lErr != nil {
		return lErr
	}
	defer listen.Close()
	for s.state {
		conn, err := listen.AcceptTCP()
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
		if err = s.scm.addConnect(id, conn); err != nil {
			log.Printf("initializeConnection id[%v] err : -2 %v\n", id, err)
			_ = conn.Close()
			return
		}
		log.Printf("initializeConnection id[%v] successfully\n", id)
	}()
}

// 认证现返回一个id和错误
func (s *Server) authentication(conn *net.TCPConn) (string, error) {
	if s.scm.count >= uint32(s.MaxConnectNum) {
		log.Printf("client connect Exceed the maximum quantity：now：[%v],max：[%v]\n", s.scm.count, s.MaxConnectNum)
		_ = write(conn, []byte("The server is busy and try again later"))
		return "", errors.New("client connect Exceed the maximum quantity")
	}
	buf, err := jeans.Unpack(conn)
	if err != nil {
		return "", err
	}
	n := bytes.IndexByte(buf, '\r')
	if n == -1 {
		_ = write(conn, []byte("0authentication fail : Illegal connection"))
		return "", errors.New("authentication fail : Illegal connection")
	}

	id := string(buf[:n])
	data := buf[:n+1]
	if id == "" {
		_ = write(conn, []byte("0authentication fail : id is null !"))
		return "", nil
	}
	mgConn, ok := s.scm.connList[id]
	if ok && mgConn.state {
		_ = write(conn, []byte("0authentication fail :The user has established a connection"))
		return id, errors.New("authentication fail :The user has established a connection")
	}

	if s.AuthenticationFunc == nil {
		_ = write(conn, []byte("1"))
		return id, nil
	}
	ok, b := s.AuthenticationFunc(id, data)
	if !ok {
		_ = write(conn, append([]byte("0"), b...))
		return id, errors.New(string(b))
	}

	err = write(conn, append([]byte("1"), b...))
	return id, err
}

func (s *Server) Close() {
	s.state = false
	s.scm.closeAllConn()
}

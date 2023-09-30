package node

import (
	"bytes"
	"errors"
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"net"
	"runtime"
	"strconv"
	"time"
)

type AuthenticationFunc func(id string, data []byte) (ok bool, reply []byte)

type Server struct {
	Id                  string
	Key                 string
	Address             string
	WorkerProcessNum    int
	Running             bool
	connectIOHandleChan chan *Context
	*ServerConnectManager
	AuthenticationFunc
	*RouteManager
	*goroutineManager
}

func NewServer(address string) *Server {
	var srv = new(Server)
	srv.Id = strconv.Itoa(int(time.Now().UnixNano()))
	srv.Address = address
	srv.WorkerProcessNum = runtime.NumCPU()
	srv.RouteManager = newRouter()
	srv.connectIOHandleChan = make(chan *Context, srv.WorkerProcessNum*2)
	return srv
}

// 开启服务
func (s *Server) ListenAndServer() error {
	//连接管理器初始化
	s.ServerConnectManager = newServerConnectManager(s.AuthenticationFunc, s.connectIOHandleChan)
	//开启工作协程池
	s.goroutineManager = newGoroutineManager(s.WorkerProcessNum, s.RouteManager, s.connectIOHandleChan, s.ServerConnectManager)
	s.goroutineManager.start()

	addr, err := net.ResolveTCPAddr("tcp", s.Address)
	if err != nil {
		return err
	}
	listen, lErr := net.ListenTCP("tcp", addr)
	if lErr != nil {
		return lErr
	}
	var conn *net.TCPConn
	for {
		conn, err = listen.AcceptTCP()
		if err != nil {
			break
		}
		s.initializeConnection(conn)
	}
	_ = listen.Close()
	return err
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
		if err = s.ServerConnectManager.addConnect(id, conn); err != nil {
			log.Printf("initializeConnection id[%v] err : -2 %v\n", id, err)
			_ = conn.Close()
			return
		}
		log.Printf("initializeConnection id[%v] successfully\n", id)
	}()
}

// 认证现返回一个id和错误
func (s *Server) authentication(conn *net.TCPConn) (string, error) {

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
	mgConn, ok := s.ServerConnectManager.connList[id]
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

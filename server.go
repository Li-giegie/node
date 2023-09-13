package node

import (
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"net"
	"runtime"
	"sync"
)

type RouterHandlerI interface {
	GetApi() uint32
	GetHandler() HandlerFunc
}

type Server struct {
	Address          string
	WorkerProcessNum int
	router           sync.Map
	TickHandle       HandlerFunc
	NoRouteHandle    HandlerFunc
}

// 添加路由
func (s *Server) AddRouter(api uint32, handler HandlerFunc) {
	s.router.Store(api, handler)
}

// 添加具有路由功能的函数
func (s *Server) AddRouterHandler(h RouterHandlerI) {
	s.router.Store(h.GetApi(), h.GetHandler())
}

// 开启服务
func (s *Server) ListenAndServer() error {
	//开启工作协程池
	startWorkerProcess(s.WorkerProcessNum, &s.router, &s.NoRouteHandle, &s.TickHandle)
	//开启连接管理器
	startServerConnectManager()

	addr, err := net.ResolveTCPAddr("tcp", s.Address)
	if err != nil {
		return err
	}
	listen, lErr := net.ListenTCP("tcp", addr)
	if lErr != nil {
		return lErr
	}
	defer listen.Close()

	for {
		conn, cErr := listen.AcceptTCP()
		if cErr != nil {
			return cErr
		}

		//接受管理一个连接
		srvConnMgmt.conn = append(srvConnMgmt.conn, newServerConnect(conn))
	}

}

func NewServer(address string) *Server {
	var srv = new(Server)
	srv.Address = address
	srv.WorkerProcessNum = runtime.NumCPU()
	srv.TickHandle = defaultTickHandle()
	srv.NoRouteHandle = defaultNoRouteHandle()
	return srv
}

type serverConnect struct {
	conn  *net.TCPConn
	state bool
	close chan struct{}
}

func newServerConnect(conn *net.TCPConn) *serverConnect {
	srvConn := new(serverConnect)
	srvConn.conn = conn
	srvConn.state = true
	srvConn.close = make(chan struct{})
	go srvConn.process()
	return srvConn
}

func (c *serverConnect) process() {
	for {
		select {
		case <-c.close:
			c.Close()
			log.Println("node read exit ---")
			return
		default:
			buf, err := jeans.Unpack(c.conn)
			if err != nil {
				c.state = false
				log.Printf("node read err :%v\n", err)
				return
			}
			msg, err := NewMessageBaseWithUnmarshal(buf)
			if err != nil {
				c.state = false
				log.Printf("node read -NewMessageBaseWithUnmarshal err :%v\n", err)
				continue
			}
			workProcess.in <- NewContext(c, msg)
			//fmt.Println("write worker success ---", len(workProcess.in), cap(workProcess.in))
		}
	}
}

func (c *serverConnect) Close() {
	c.state = false
	_ = c.conn.Close()
	close(c.close)
}

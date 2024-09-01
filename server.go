package node

import (
	"context"
	"errors"
	"github.com/Li-giegie/node/common"
	"log"
	"net"
	"sync"
	"time"
)

type StateType uint8

const (
	StateType_Close StateType = iota
	StateType_Listen
	StateType_Err
)

type Server struct {
	id uint16 // 唯一标识
	// 最大连接数 <=0 不限制 默认0
	MaxConns int
	// InitSessionTimeout 初始化连接，并在限定时间得到节点id，默认6s
	InitSessionTimeout time.Duration
	State              StateType
	Conns              *Conns
	msgReceiver        *common.MsgReceiver
	Router             *common.RouteTable
	handler            Handler
	listen             net.Listener
}

// NewServer 创建一个Server类型的节点
func NewServer(l net.Listener, id uint16) *Server {
	srv := new(Server)
	srv.id = id
	srv.listen = l
	srv.InitSessionTimeout = time.Second * 6
	srv.Conns = newConns()
	srv.msgReceiver = common.NewMsgReceiver(1024)
	srv.Router = common.NewRouter()
	return srv
}

func (s *Server) SetMsgReceiver(n int) {
	s.msgReceiver = common.NewMsgReceiver(n)
}
func (s *Server) Serve(h Handler) error {
	s.State = StateType_Listen
	s.handler = h
	i := int64(0)
	for {
		if s.MaxConns > 0 && s.Conns.Len() >= s.MaxConns {
			if i <= 10 {
				i++
			}
			time.Sleep(time.Second * time.Duration(i))
			log.Println("Connection pool overflow, exceeding maximum number of connections")
			continue
		}
		i = 0
		conn, err := s.listen.Accept()
		if err != nil {
			return s.checkErr(err)
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	var err error
	connInit := new(ConnInitializer)
	if err = connInit.ReceptionWithTimeout(s.InitSessionTimeout, conn); err != nil {
		_ = conn.Close()
		return
	}
	c := common.NewConn(s.id, connInit.LocalId, conn, s.msgReceiver, s.Conns, s.Router, s.handler)
	if connInit.RemoteId != s.id {
		connInit.code = authCode_ridErr
		_ = connInit.Send(conn)
		_ = conn.Close()
		return
	}
	if connInit.LocalId == s.id || !s.Conns.add(connInit.LocalId, c) {
		connInit.code = authCode_nodeExist
		_ = connInit.Send(conn)
		_ = conn.Close()
		return
	}
	connInit.code = authCode_success
	if err = connInit.Send(conn); err != nil {
		_ = conn.Close()
		return
	}
	go func() {
		c.Serve()
		s.Conns.del(c.RemoteId())
		_ = c.Close()
	}()
	s.handler.Connection(c)
}

var nodeEqErr = errors.New("node ID cannot be the same as the server node ID")

// BindBridge 桥接一个域,使用一个客户端连接到其他节点并绑定到当前节点形成一个大的域
func (s *Server) BindBridge(bd BridgeNode) error {
	if s.id == bd.RemoteId() {
		return nodeEqErr
	}
	conn := common.NewConn(s.id, bd.RemoteId(), bd.Conn(), s.msgReceiver, s.Conns, s.Router, s.handler)
	if !s.Conns.add(conn.RemoteId(), conn) {
		return nodeExistErr
	}
	go func() {
		conn.Serve()
		s.Conns.del(conn.RemoteId())
		_ = conn.Close()
		bd.Disconnection()
	}()
	s.handler.Connection(conn)
	return nil
}

// Request 请求
func (s *Server) Request(ctx context.Context, dst uint16, data []byte) ([]byte, error) {
	conn, ok := s.findConn(dst)
	if !ok {
		return nil, common.DEFAULT_ErrConnNotExist
	}
	return conn.Forward(ctx, dst, data)
}

func (s *Server) WriteTo(dst uint16, data []byte) (int, error) {
	conn, ok := s.findConn(dst)
	if !ok {
		return 0, common.DEFAULT_ErrConnNotExist
	}
	return conn.WriteTo(dst, data)
}

func (s *Server) findConn(dst uint16) (conn common.Conn, exists bool) {
	conn, exists = s.Conns.GetConn(dst)
	if exists {
		return
	}
	routes := s.Router.GetDstRoutes(dst)
	for i := 0; i < len(routes); i++ {
		conn, exists = s.Conns.GetConn(routes[i].Next)
		if exists {
			return
		}
	}
	return nil, false
}

func (s *Server) Id() uint16 {
	return s.id
}

func (s *Server) checkErr(err error) error {
	if s.State == StateType_Close {
		return nil
	}
	s.State = StateType_Err
	return err
}

func (s *Server) Close() error {
	s.State = StateType_Close
	return s.listen.Close()
}

// ListenTCP 侦听一个本地TCP端口,并创建服务节点
func ListenTCP(addr string, id uint16) (*Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewServer(l, id), nil
}

type Conns struct {
	m map[uint16]common.Conn
	l sync.RWMutex
}

func newConns() *Conns {
	return &Conns{
		m: make(map[uint16]common.Conn),
		l: sync.RWMutex{},
	}
}

func (s *Conns) add(id uint16, conn *common.Connect) bool {
	s.l.Lock()
	v, exist := s.m[id]
	if !exist || v.State() != common.ConnStateTypeOnConnect {
		s.m[id] = conn
		exist = true
	} else {
		exist = false
	}
	s.l.Unlock()
	return exist
}

func (s *Conns) del(id uint16) {
	s.l.Lock()
	delete(s.m, id)
	s.l.Unlock()
}

func (s *Conns) GetConn(id uint16) (common.Conn, bool) {
	s.l.RLock()
	v, ok := s.m[id]
	s.l.RUnlock()
	return v, ok
}

func (s *Conns) GetConns() []common.Conn {
	s.l.RLock()
	result := make([]common.Conn, 0, len(s.m))
	for _, conn := range s.m {
		result = append(result, conn)
	}
	s.l.RUnlock()
	return result
}

func (s *Conns) Len() (n int) {
	s.l.RLock()
	n = len(s.m)
	s.l.RUnlock()
	return
}

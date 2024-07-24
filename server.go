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

// ServerStateType 服务状态
type ServerStateType uint8

const (
	ServerStateTypeClose ServerStateType = iota
	ServerStateTypeListen
	ServerStateTypeErr
)

type Server interface {
	// Serve 阻塞启动服务，node 连接生命周期接口
	Serve() error
	//GetConn 获取一个连接
	GetConn(id uint16) (common.Conn, bool)
	// GetConns 获取所有连接
	GetConns() []common.Conn
	// State 获取服务状态
	State() ServerStateType
	// Close 关闭服务
	Close() error
	// Id 获取服务Id
	Id() uint16
	// Bind 同步阻塞调用，绑定一个外部连接，通常用于其他域互联形成一个域
	Bind(u ExternalDomainNode) error
	// Request 发起一个请求
	Request(ctx context.Context, dst uint16, data []byte) ([]byte, error)
	WriteTo(dst uint16, data []byte) (int, error)
	common.Router
}

type SrvOption func(s *server)

type server struct {
	id       uint16 // 唯一标识
	MaxConns int
	state    ServerStateType
	net.Listener
	*connections
	*common.MsgReceiver
	*common.MsgPool
	common.Router
	Handler
}

// NewServer 创建一个Server类型的节点
func NewServer(l net.Listener, id uint16, h Handler, opts ...SrvOption) Server {
	srv := new(server)
	srv.id = id
	srv.Listener = l
	srv.Handler = h
	srv.connections = newConns()
	srv.MsgPool = common.NewMsgPool(1024)
	srv.MsgReceiver = common.NewMsgReceiver(1024)
	srv.Router = common.NewRouter()
	for _, opt := range opts {
		opt(srv)
	}
	return srv
}

func (s *server) State() ServerStateType {
	return s.state
}

func (s *server) Serve() error {
	s.state = ServerStateTypeListen
	i := int64(0)
	for {
		if s.MaxConns > 0 && s.connections.Len() >= s.MaxConns {
			if i <= 10 {
				i++
			}
			time.Sleep(time.Second * time.Duration(i))
			log.Println("Connection pool overflow, exceeding maximum number of connections")
			continue
		}
		i = 0
		conn, err := s.Accept()
		if err != nil {
			return s.checkErr(err)
		}
		go s.HandleConn(conn)
	}
}

func (s *server) HandleConn(c net.Conn) {
	remoteId, err := s.Handler.Init(c)
	if err != nil {
		return
	}
	conn := common.NewConn(s.id, remoteId, c, s, s.connections, s.Router)
	if conn.RemoteId() == s.id || !s.Add(conn.RemoteId(), conn) {
		s.Handler.Disconnect(conn.RemoteId(), common.DEFAULT_ErrAuthIdExist)
		_ = conn.WriteMsg(&common.Message{
			Type:   common.MsgType_PushErrAuthFailIdExist,
			SrcId:  s.id,
			DestId: conn.RemoteId(),
		})
		_ = c.Close()
		return
	}
	go func() {
		conn.Serve(s.Handler)
		s.connections.Del(conn.RemoteId())
		_ = c.Close()
	}()
}

var NodeExist = errors.New("node id exist")

func (s *server) Bind(u ExternalDomainNode) error {
	if s.id == u.RemoteId() {
		return NodeExist
	}
	conn := common.NewConn(s.id, u.RemoteId(), u.Conn(), s, s.connections, s.Router)
	if !s.connections.Add(conn.RemoteId(), conn) {
		return NodeExist
	}
	conn.Serve(s)
	s.connections.Del(conn.RemoteId())
	_ = conn.Close()
	return nil
}

func (s *server) Request(ctx context.Context, dst uint16, data []byte) ([]byte, error) {
	conn, ok := s.GetConn(dst)
	if ok {
		return conn.Request(ctx, data)
	}
	rInfo := s.GetDstRoutes(dst)
	for i := 0; i < len(rInfo); i++ {
		conn, ok = s.GetConn(rInfo[i].Next)
		if ok {
			return conn.Forward(ctx, dst, data)
		}
	}
	return nil, common.DEFAULT_ErrConnNotExist
}

func (s *server) WriteTo(dst uint16, data []byte) (int, error) {
	conn, ok := s.GetConn(dst)
	if ok {
		return conn.Write(data)
	}
	rInfo := s.GetDstRoutes(dst)
	for i := 0; i < len(rInfo); i++ {
		conn, ok = s.GetConn(rInfo[i].Next)
		if ok {
			return conn.WriteTo(dst, data)
		}
	}
	return 0, common.DEFAULT_ErrConnNotExist
}

func (s *server) Id() uint16 {
	return s.id
}

func (s *server) checkErr(err error) error {
	if s.state == ServerStateTypeClose {
		return nil
	}
	s.state = ServerStateTypeErr
	return err
}

func (s *server) Close() error {
	s.state = ServerStateTypeClose
	return s.Listener.Close()
}

// WithSrvMaxConns 最大连接数 > 0 有效
func WithSrvMaxConns(n int) SrvOption {
	return func(s *server) {
		s.MaxConns = n
	}
}

// WIthSrvMsgPoolSize 消息池容量，消息在从池子中创建和销毁，这一行为是考虑到GC压力
func WIthSrvMsgPoolSize(n int) SrvOption {
	return func(s *server) {
		s.MsgPool = common.NewMsgPool(n)
	}
}

// WithSrvMsgReceivePoolSize 消息接收池容量，消息接收每次创建的Channel从池子中创建和销毁，这一行为是考虑到GC压力
func WithSrvMsgReceivePoolSize(n int) SrvOption {
	return func(s *server) {
		s.MsgReceiver = common.NewMsgReceiver(n)
	}
}

// ListenTCP 侦听一个本地TCP端口,并创建服务节点
func ListenTCP(id uint16, addr string, h Handler, opts ...SrvOption) (Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewServer(l, id, h, opts...), nil
}

type connections struct {
	m map[uint16]common.Conn
	l sync.RWMutex
}

func newConns() *connections {
	return &connections{
		m: make(map[uint16]common.Conn),
		l: sync.RWMutex{},
	}
}

func (s *connections) Add(id uint16, conn *common.Connect) bool {
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

func (s *connections) Del(id uint16) {
	s.l.Lock()
	delete(s.m, id)
	s.l.Unlock()
}

func (s *connections) GetConn(id uint16) (common.Conn, bool) {
	s.l.RLock()
	v, ok := s.m[id]
	s.l.RUnlock()
	return v, ok
}

func (s *connections) GetConns() []common.Conn {
	s.l.RLock()
	result := make([]common.Conn, 0, len(s.m))
	for _, conn := range s.m {
		result = append(result, conn)
	}
	s.l.RUnlock()
	return result
}

func (s *connections) Len() (n int) {
	s.l.RLock()
	n = len(s.m)
	s.l.RUnlock()
	return
}

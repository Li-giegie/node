package node

import (
	"context"
	"errors"
	"github.com/Li-giegie/node/common"
	"log"
	"net"
	"time"
)

type Server interface {
	// Serve 阻塞启动服务，node 连接生命周期接口
	Serve() error
	// GetConn 获取一个连接
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
	Send(dst uint16, data []byte) error
	common.Router
}

// ServerStateType 服务状态
type ServerStateType uint8

const (
	// ServerStateTypeClose 关闭或未开启
	ServerStateTypeClose ServerStateType = iota
	// ServerStateTypeListen 开启侦听
	ServerStateTypeListen
	// ServerStateTypeErr 错误
	ServerStateTypeErr
)

type SrvOption func(s *server)

type server struct {
	id       uint16 // 唯一标识
	MaxConns int
	//MaxMsgLen uint32
	state ServerStateType
	net.Listener
	*common.Conns
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
	srv.Conns = common.NewConns()
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
		if s.MaxConns > 0 && s.Conns.Len() >= s.MaxConns {
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
	conn := common.NewConn(s.id, remoteId, c, s, s.Conns, s.Router)
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
		s.Conns.Del(conn.RemoteId())
		_ = c.Close()
	}()
}

var NodeExist = errors.New("node id exist")

func (s *server) Bind(u ExternalDomainNode) error {
	if s.id == u.RemoteId() {
		return NodeExist
	}
	conn := common.NewConn(s.id, u.RemoteId(), u.Conn(), s, s.Conns, s.Router)
	if !s.Conns.Add(conn.RemoteId(), conn) {
		return NodeExist
	}
	conn.Serve(s)
	s.Conns.Del(conn.RemoteId())
	_ = conn.Close()
	return nil
}

func (s *server) Request(ctx context.Context, dst uint16, data []byte) ([]byte, error) {
	log.Println("dst", dst)
	conn, ok := s.GetConn(dst)
	if ok {
		return conn.Request(ctx, data)
	}
	rInfo := s.GetDstRoutes(dst)
	for i := 0; i < len(rInfo); i++ {
		conn, ok = s.GetConn(rInfo[i].Next)
		if ok {
			return conn.Request(ctx, data)
		}
	}
	return nil, common.DEFAULT_ErrConnNotExist
}

func (s *server) Send(dst uint16, data []byte) error {
	conn, ok := s.GetConn(dst)
	if ok {
		return conn.Send(data)
	}
	rInfo := s.GetDstRoutes(dst)
	for i := 0; i < len(rInfo); i++ {
		conn, ok = s.GetConn(rInfo[i].Next)
		if ok {
			return conn.Send(data)
		}
	}
	return common.DEFAULT_ErrConnNotExist
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

// ListenTCP 侦听一个本地TCP端口,并创建服务节点
func ListenTCP(id uint16, addr string, h Handler, opts ...SrvOption) (Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewServer(l, id, h, opts...), nil
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

func WithSrvRouter(route common.Router) SrvOption {
	return func(s *server) {
		s.Router = route
	}
}

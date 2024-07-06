package node

import (
	"errors"
	"github.com/Li-giegie/node/common"
	"log"
	"net"
	"time"
)

type Server interface {
	// Serve 阻塞启动服务，node 连接生命周期接口
	Serve(node Node) error
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

type SrvOption func(s *server) error

type server struct {
	id        uint16 // 唯一标识
	MaxConns  int
	MaxMsgLen uint32
	state     ServerStateType
	net.Listener
	*common.Conns
	*common.MsgReceiver
	*common.MsgPool
	*common.RouteTable
}

// NewServer 创建一个Server类型的节点
func NewServer(l net.Listener, id uint16, opts ...SrvOption) (Server, error) {
	srv := new(server)
	srv.id = id
	srv.Listener = l
	srv.Conns = common.NewConns()
	srv.MsgPool = common.NewMsgPool(1024)
	srv.MsgReceiver = common.NewMsgReceiver(1024)
	for _, opt := range opts {
		if err := opt(srv); err != nil {
			return nil, err
		}
	}
	return srv, nil
}

func (s *server) State() ServerStateType {
	return s.state
}

func (s *server) Serve(node Node) error {
	if node == nil {
		_ = s.Close()
		return errors.New("err: Handler can not be null ")
	}
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
		go s.HandleConn(conn, node)
	}
}

func (s *server) HandleConn(c net.Conn, node Node) {
	conn, err := common.NewConn(s.id, c, s, s.Conns, node, s.RouteTable)
	if err != nil {
		return
	}
	if conn.RemoteId() == s.id || !s.Add(conn.RemoteId(), conn) {
		node.Disconnect(conn.RemoteId(), common.DEFAULT_ErrAuthIdExist)
		_ = conn.WriteMsg(&common.Message{
			Type:   common.MsgType_PushErrAuthFailIdExist,
			SrcId:  s.id,
			DestId: conn.RemoteId(),
		})
		_ = c.Close()
		return
	}
	go func() {
		conn.Serve()
		s.Conns.Del(conn.RemoteId())
		_ = c.Close()
	}()
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
func ListenTCP(id uint16, addr string, opts ...SrvOption) (Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewServer(l, id, opts...)
}

// WithSrvMaxConns 最大连接数 > 0 有效
func WithSrvMaxConns(n int) SrvOption {
	return func(s *server) error {
		s.MaxConns = n
		return nil
	}
}

// WithSrvMaxMsgLen 最大消息接收长度 > 0 <= 256*256*256-1 时有效 最大值3个字节范围的正整数
func WithSrvMaxMsgLen(n int) SrvOption {
	return func(s *server) error {
		if n < 0 || n > 0x00FFFFFF {
			return errors.New("err: MaxMsgLen > 0 < 0x00FFFFFF,3byte")
		}
		s.MaxMsgLen = uint32(n)
		return nil
	}
}

// WIthSrvMsgPoolSize 消息池容量，消息在从池子中创建和销毁，这一行为是考虑到GC压力
func WIthSrvMsgPoolSize(n int) SrvOption {
	return func(s *server) error {
		s.MsgPool = common.NewMsgPool(n)
		return nil
	}
}

// WithSrvMsgReceivePoolSize 消息接收池容量，消息接收每次创建的Channel从池子中创建和销毁，这一行为是考虑到GC压力
func WithSrvMsgReceivePoolSize(n int) SrvOption {
	return func(s *server) error {
		s.MsgReceiver = common.NewMsgReceiver(n)
		return nil
	}
}

func WithSrvRouter(enable bool, route ...*common.RouteTable) SrvOption {
	return func(s *server) error {
		if len(route) > 0 && route[0] != nil {
			s.RouteTable = route[0]
		} else if enable {
			s.RouteTable = common.NewRouter()
		}
		return nil
	}
}

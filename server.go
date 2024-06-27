package node

import (
	"errors"
	"github.com/Li-giegie/node/common"
	"log"
	"net"
	"time"
)

type Server interface {
	Serve(node Node) error
	GetConn(id uint16) (common.Conn, bool)
	GetConns() []common.Conn
	State() ServerStateType
	Close() error
	Id() uint16
}

type ServerStateType uint8

const (
	ServerStateTypeClose ServerStateType = iota
	ServerStateTypeListen
	ServerStateTypeErr
)

type SrvOptions func(s *server) error

type server struct {
	id        uint16 // 唯一标识
	MaxConns  int
	MaxMsgLen uint32
	state     ServerStateType
	net.Listener
	*common.Conns
	*common.MsgReceiver
	*common.MsgPool
}

// NewServer 创建一个Server类型的节点
func NewServer(l net.Listener, id uint16, opts ...SrvOptions) (Server, error) {
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
	conn, err := common.NewConn(s.id, c, s.MsgPool, s.MsgReceiver, s, node)
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
		conn.Serve(node)
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

func ListenTCP(id uint16, addr string, opts ...SrvOptions) (Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewServer(l, id, opts...)
}

// WithSrvMaxConns 最大连接数 > 0 有效
func WithSrvMaxConns(n int) SrvOptions {
	return func(s *server) error {
		s.MaxConns = n
		return nil
	}
}

// WithSrvMaxMsgLen 最大消息接收长度 > 0 <= 256*256*256-1 时有效 最大值3个字节范围的正整数
func WithSrvMaxMsgLen(n int) SrvOptions {
	return func(s *server) error {
		if n < 0 || n > 0x00FFFFFF {
			return errors.New("err: MaxMsgLen > 0 < 0x00FFFFFF,3byte")
		}
		s.MaxMsgLen = uint32(n)
		return nil
	}
}

// WIthSrvMsgPoolSize 消息在从池子中创建和销毁，这一行为是考虑到GC压力
func WIthSrvMsgPoolSize(n int) SrvOptions {
	return func(s *server) error {
		s.MsgPool = common.NewMsgPool(n)
		return nil
	}
}

// WithSrvMsgReceivePoolSize 消息接收每次创建的Channel从池子中创建和销毁，这一行为是考虑到GC压力
func WithSrvMsgReceivePoolSize(n int) SrvOptions {
	return func(s *server) error {
		s.MsgReceiver = common.NewMsgReceiver(n)
		return nil
	}
}

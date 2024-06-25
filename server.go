package node

import (
	"errors"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/utils"
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
	// SetMaxConns 最大连接数 > 0 有效
	SetMaxConns(n int)
	// SetMaxMsgLen 最大消息接收长度 > 0 <= 256*256*256-1 时有效 最大值3个字节范围的正整数
	SetMaxMsgLen(n int) error
	// SetMsgPoolSize 消息在从池子中创建和销毁，这一步骤是考虑到GC压力
	SetMsgPoolSize(n int)
	// SetMsgReceivePoolSize 消息接收每次创建的Channel从池子中创建和销毁，这一步骤是考虑到GC压力
	SetMsgReceivePoolSize(n int)
	Id() uint16
}

type ServerStateType uint8

const (
	ServerStateTypeClose ServerStateType = iota
	ServerStateTypeListen
	ServerStateTypeErr
)

type server struct {
	id        uint16 // 唯一标识
	MaxConns  int
	MaxMsgLen uint32
	state     ServerStateType
	Node
	net.Listener
	*common.Conns
	*common.MsgReceiver
	*common.MsgPool
}

func (s *server) SetMaxConns(n int) {
	s.MaxConns = n
}

func (s *server) SetMaxMsgLen(n int) error {
	if n < 0 || n > 0x00FFFFFF {
		return errors.New("err: MaxMsgLen > 0 < 0x00FFFFFF,3byte")
	}
	s.MaxMsgLen = uint32(n)
	return nil
}

func (s *server) SetMsgPoolSize(n int) {
	s.MsgPool = common.NewMsgPool(n)
}

func (s *server) SetMsgReceivePoolSize(n int) {
	s.MsgReceiver = common.NewMsgReceiver(n)
}

// NewServer 创建一个Server类型的节点
func NewServer(l net.Listener, id uint16) Server {
	srv := new(server)
	srv.id = id
	srv.Listener = l
	srv.Conns = common.NewConns()
	srv.MsgPool = common.NewMsgPool(1024)
	srv.MsgReceiver = common.NewMsgReceiver(1024)
	srv.state = ServerStateTypeListen
	return srv
}

func (s *server) State() ServerStateType {
	return s.state
}

func (s *server) Serve(h Node) error {
	if h == nil {
		_ = s.Close()
		return errors.New("err: Handler can not be null ")
	}
	s.Node = h
	i := int64(1)
	d := time.Second
	for {
		if utils.CountSleep(s.MaxConns > 0 && s.Conns.Len() >= s.MaxConns, i, d) {
			if i <= 10 {
				i++
			}
			log.Println("Connection pool overflow, exceeding maximum number of connections")
			continue
		}
		i = 1
		conn, err := s.Accept()
		if err != nil {
			return s.checkErr(err)
		}
		go s.HandleConn(conn)
	}
}

func (s *server) HandleConn(c net.Conn) {
	conn, err := common.NewConn(s.id, c, s.MsgPool, s.MsgReceiver, s, s.Node)
	if err != nil {
		return
	}
	if s.Add(conn.RemoteId(), conn) {
		go func() {
			conn.Serve(s.Node)
			s.Conns.Del(conn.RemoteId())
			_ = c.Close()
		}()
		return
	}
	s.Conns.Del(conn.RemoteId())
	s.Node.Disconnect(conn.RemoteId(), common.DEFAULT_ErrAuth)
	_ = conn.WriteMsg(&common.Message{
		Type:   common.MsgType_PushErrAuthFail,
		SrcId:  s.id,
		DestId: conn.RemoteId(),
	})
	_ = c.Close()
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

func ListenTCP(id uint16, addr string) (Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewServer(l, id), nil
}

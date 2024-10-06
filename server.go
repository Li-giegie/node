package node

import (
	"context"
	"errors"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/utils"
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
	MaxConns        int
	MaxMsgLen       uint32
	WriterQueueSize int
	State           StateType
	Conns           *Conns
	revChan         map[uint32]chan *common.Message
	lock            *sync.Mutex
	Router          *common.RouteTable
	handler         Handler
	listen          net.Listener
	counter         uint32
	ReaderBufSize   int
	WriterBufSize   int
	*Identity
}

// NewServer 创建一个Server类型的节点
func NewServer(l net.Listener, id *Identity) *Server {
	if id == nil {
		panic("BasicAuth can't be empty")
	}
	srv := new(Server)
	srv.Identity = id
	srv.listen = l
	srv.Conns = newConns()
	srv.revChan = make(map[uint32]chan *common.Message)
	srv.lock = new(sync.Mutex)
	srv.ReaderBufSize = 4096
	srv.WriterBufSize = 4096
	srv.MaxMsgLen = 0xffffff
	srv.WriterQueueSize = 1024
	srv.Router = common.NewRouter()
	return srv
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
	rid, key, err := defaultBasicReq.Receive(conn, s.AccessTimeout)
	if err != nil {
		_ = conn.Close()
		return
	}
	if !utils.BytesEqual(s.AccessKey, key) {
		_ = defaultBasicResp.Send(conn, 0, false, "error: AccessKey invalid")
		_ = conn.Close()
		return
	}
	lid := s.Identity.Id
	c := common.NewConn(lid, rid, conn, s.revChan, s.lock, s.Conns, s.Router, s.handler, &s.counter, s.ReaderBufSize, s.WriterBufSize, s.WriterQueueSize, s.MaxMsgLen)
	if rid == lid || !s.Conns.add(rid, c) {
		_ = defaultBasicResp.Send(conn, 0, false, "error: id already exists")
		_ = conn.Close()
		return
	}
	if err = defaultBasicResp.Send(conn, lid, true, ""); err != nil {
		_ = conn.Close()
		s.Conns.del(c.RemoteId())
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
var errNodeExist = errors.New("node already exists")

// BindBridge 桥接一个域,使用一个客户端连接到其他节点并绑定到当前节点形成一个大的域
func (s *Server) BindBridge(bd BridgeNode) error {
	if s.Identity.Id == bd.RemoteId() {
		return nodeEqErr
	}
	conn := common.NewConn(s.Identity.Id, bd.RemoteId(), bd.Conn(), s.revChan, s.lock, s.Conns, s.Router, s.handler, &s.counter, s.ReaderBufSize, s.WriterBufSize, s.WriterQueueSize, s.MaxMsgLen)
	if !s.Conns.add(conn.RemoteId(), conn) {
		return errNodeExist
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
func (s *Server) Request(ctx context.Context, dst uint32, data []byte) ([]byte, error) {
	conn, ok := s.findConn(dst)
	if !ok {
		return nil, common.DEFAULT_ErrConnNotExist
	}
	return conn.Forward(ctx, dst, data)
}

func (s *Server) WriteTo(dst uint32, data []byte) (int, error) {
	conn, ok := s.findConn(dst)
	if !ok {
		return 0, common.DEFAULT_ErrConnNotExist
	}
	return conn.WriteTo(dst, data)
}

func (s *Server) findConn(dst uint32) (conn common.Conn, exists bool) {
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

func (s *Server) Id() uint32 {
	return s.Identity.Id
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
func ListenTCP(addr string, auth *Identity) (*Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewServer(l, auth), nil
}

type Conns struct {
	m map[uint32]common.Conn
	l sync.RWMutex
}

func newConns() *Conns {
	return &Conns{
		m: make(map[uint32]common.Conn),
		l: sync.RWMutex{},
	}
}

func (s *Conns) add(id uint32, conn *common.Connect) bool {
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

func (s *Conns) del(id uint32) {
	s.l.Lock()
	delete(s.m, id)
	s.l.Unlock()
}

func (s *Conns) GetConn(id uint32) (common.Conn, bool) {
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

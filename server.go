package node

import (
	"context"
	"errors"
	nodeNet "github.com/Li-giegie/node/net"
	"io"
	"net"
	"sync"
	"time"
)

// SrvConf Server 配置对象
type SrvConf struct {
	// 当前节点凭证
	*Identity
	// 收发消息最大长度
	MaxMsgLen uint32
	// 消息并发写入队列，队列长度
	WriterQueueSize int
	// 读缓存区大小
	ReaderBufSize int
	// 写缓冲区大小
	WriterBufSize int
	// 最大连接数
	MaxConns int
	// 超过最大连接数时，进入休眠的最大时间，按照步长递增
	MaxListenSleepTime time.Duration
	// 超过限制连接数量，递增休眠步长，直到达到最大休眠时长后停止递增
	ListenStepTime time.Duration
	// 连接建立认证通过回调
	OnConnection func(conn Conn) `yaml:"-" json:"-"`
	// 收到消息回调
	OnMessage func(ctx Context) `yaml:"-" json:"-"`
	// 收到自定义类型的消息回调
	OnCustomMessage func(ctx CustomContext) `yaml:"-" json:"-"`
	// 连接被关闭调用
	OnClose func(id uint32, err error) `yaml:"-" json:"-"`
}

type Server struct {
	recvChan map[uint32]chan *nodeNet.Message
	recvLock sync.Mutex
	Router   *nodeNet.RouteTable
	listen   net.Listener
	counter  uint32
	*SrvConf
	*ConnManager
}

// NewServer 创建一个Server类型的节点
func NewServer(l net.Listener, c *SrvConf) *Server {
	srv := new(Server)
	srv.SrvConf = c
	srv.ConnManager = NewConnManager()
	srv.Router = nodeNet.NewRouter()
	srv.recvChan = make(map[uint32]chan *nodeNet.Message)
	srv.listen = nodeNet.NewLimitListener(l, c.MaxConns, c.MaxListenSleepTime, c.ListenStepTime, srv.ConnManager)
	return srv
}

func (s *Server) Serve() error {
	for {
		conn, err := s.listen.Accept()
		if err != nil {
			if _, ok := err.(nodeNet.ErrClosedListen); ok {
				return nil
			}
			return err
		}
		go s.handleAuthenticate(conn)
	}
}

func (s *Server) handleAuthenticate(conn net.Conn) {
	rid, key, err := defaultBasicReq.Receive(conn, s.AuthTimeout)
	if err != nil {
		_ = conn.Close()
		return
	}
	if !BytesEqual(s.AuthKey, key) {
		_ = defaultBasicResp.Send(conn, 0, false, "error: AccessKey invalid")
		_ = conn.Close()
		return
	}
	lid := s.Identity.Id
	c := nodeNet.NewConn(lid, rid, conn, s.recvChan, &s.recvLock, &s.counter, s.ReaderBufSize, s.WriterBufSize, s.WriterQueueSize, s.MaxMsgLen)
	if rid == lid || !s.ConnManager.Add(rid, c) {
		_ = defaultBasicResp.Send(conn, 0, false, "error: id already exists")
		_ = conn.Close()
		return
	}
	if err = defaultBasicResp.Send(conn, lid, true, ""); err != nil {
		_ = conn.Close()
		s.ConnManager.Remove(c.RemoteId())
		return
	}
	s.startConn(c)
}

func (s *Server) startConn(c *nodeNet.Connect) {
	if s.OnConnection != nil {
		s.OnConnection(c)
	}
	hBuf := make([]byte, nodeNet.MsgHeaderLen)
	for {
		msg, err := c.ReadMsg(hBuf)
		if err != nil {
			if c.IsClosed || errors.Is(err, io.EOF) {
				err = nil
			}
			_ = c.Close()
			s.ConnManager.Remove(c.RemoteId())
			if s.OnClose != nil {
				s.OnClose(c.RemoteId(), err)
			}
			return
		}
		if msg.DestId != s.Identity.Id {
			// 优先转发到直连连接
			if dstConn, exist := s.ConnManager.Get(msg.DestId); exist {
				if _, err = dstConn.WriteMsg(msg); err == nil {
					continue
				}
			}
			nextList := s.Router.GetDstRoutes(msg.DestId)
			success := false
			for i := 0; i < len(nextList); i++ {
				dstConn, exist := s.ConnManager.Get(nextList[i].Next)
				if exist {
					if _, err = dstConn.WriteMsg(msg); err == nil {
						success = true
						break
					}
				}
				s.Router.DeleteRoute(msg.DestId, nextList[i].Next, nextList[i].ParentNode, nextList[i].Hop)
			}
			if success {
				continue
			}
			if len(nextList) > 0 {
				s.Router.DeleteRouteAll(msg.DestId)
			}
			// 本地节点、路由均为目的节点，返回错误
			msg.Type = nodeNet.MsgType_ReplyErrConnNotExist
			msg.DestId = msg.SrcId
			msg.SrcId = s.Identity.Id
			_, _ = c.WriteMsg(msg)
			continue
		}
		switch msg.Type {
		case nodeNet.MsgType_Send:
			s.OnMessage(&connContext{Message: msg, Connect: c})
		case nodeNet.MsgType_Reply, nodeNet.MsgType_ReplyErr, nodeNet.MsgType_ReplyErrConnNotExist, nodeNet.MsgType_ReplyErrLenLimit, nodeNet.MsgType_ReplyErrCheckSum:
			s.recvLock.Lock()
			ch, ok := s.recvChan[msg.Id]
			if ok {
				ch <- msg
				delete(s.recvChan, msg.Id)
			}
			s.recvLock.Unlock()
		default:
			if s.OnCustomMessage != nil {
				s.OnCustomMessage(&connContext{Message: msg, Connect: c})
			}
		}
	}
}

var nodeEqErr = errors.New("node ID cannot be the same as the server node ID")
var errNodeExist = errors.New("node already exists")

// BindBridge 桥接一个域,使用一个客户端连接到其他节点并绑定到当前节点形成一个大的域
func (s *Server) BindBridge(conn net.Conn, remoteAuthKey []byte, timeout time.Duration) (rid uint32, err error) {
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	if err = defaultBasicReq.Send(conn, s.Identity.Id, remoteAuthKey); err != nil {
		return 0, err
	}
	rid, permit, msg, err := defaultBasicResp.Receive(conn, timeout)
	if err != nil {
		return rid, err
	}
	if !permit {
		return rid, errors.New(msg)
	}
	if rid == s.Identity.Id {
		return rid, nodeEqErr
	}
	c := nodeNet.NewConn(s.Identity.Id, rid, conn, s.recvChan, &s.recvLock, &s.counter, s.ReaderBufSize, s.WriterBufSize, s.WriterQueueSize, s.MaxMsgLen)
	if !s.ConnManager.Add(rid, c) {
		return rid, errNodeExist
	}
	go s.startConn(c)
	return rid, nil
}

// Request 请求
func (s *Server) Request(ctx context.Context, dst uint32, data []byte) ([]byte, error) {
	conn, ok := s.FindConn(dst)
	if !ok {
		return nil, nodeNet.DEFAULT_ErrConnNotExist
	}
	return conn.Forward(ctx, dst, data)
}

func (s *Server) WriteTo(dst uint32, data []byte) (int, error) {
	conn, ok := s.FindConn(dst)
	if !ok {
		return 0, nodeNet.DEFAULT_ErrConnNotExist
	}
	return conn.WriteTo(dst, data)
}

func (s *Server) FindConn(dst uint32) (conn Conn, exists bool) {
	conn, exists = s.ConnManager.Get(dst)
	if exists {
		return
	}
	routes := s.Router.GetDstRoutes(dst)
	for i := 0; i < len(routes); i++ {
		conn, exists = s.ConnManager.Get(routes[i].Next)
		if exists {
			return
		}
	}
	return nil, false
}

func (s *Server) Id() uint32 {
	return s.Identity.Id
}

func (s *Server) Close() error {
	return s.listen.Close()
}

package node

import (
	"context"
	"errors"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
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
}

// NewServer 创建一个Server类型的节点
func NewServer(l net.Listener, c SrvConf) iface.Server {
	srv := new(Server)
	srv.SrvConf = &c
	srv.Router = nodeNet.NewRouteTable()
	srv.ConnManager = nodeNet.NewConnManager()
	srv.recvChan = make(map[uint32]chan *message.Message)
	srv.listen = nodeNet.NewLimitListener(l, c.MaxConns, c.MaxListenSleepTime, c.ListenStepTime, srv.ConnManager)
	return srv
}

type Server struct {
	recvChan         map[uint32]chan *message.Message
	recvLock         sync.Mutex
	listen           net.Listener
	counter          uint32
	OnConnections    []func(conn iface.Conn)
	OnMessages       []func(ctx iface.Context)
	OnCustomMessages []func(ctx iface.Context)
	OnNoRouteMessage []func(ctx iface.Context)
	OnCloseds        []func(conn iface.Conn, err error) // 连接被关闭调用
	iface.Router
	iface.ConnManager
	*SrvConf
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
	rid, key, nt, err := defaultBasicReq.Receive(conn, s.AuthTimeout)
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
	c := nodeNet.NewConn(lid, rid, conn, s.recvChan, &s.recvLock, &s.counter, s.ReaderBufSize, s.WriterBufSize, s.WriterQueueSize, s.MaxMsgLen, uint8(nt))
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
	s.handleOnConnections(c)
	for {
		msg, err := c.ReadMsg()
		if err != nil {
			if c.IsClosed() || errors.Is(err, io.EOF) {
				err = nil
			}
			_ = c.Close()
			s.ConnManager.Remove(c.RemoteId())
			s.handleOnClosed(c, err)
			return
		}
		// 当前节点消息
		if msg.DestId == s.Identity.Id {
			switch msg.Type {
			case message.MsgType_Send:
				s.handleOnMessages(nodeNet.NewContext(c, msg, true))
			case message.MsgType_Reply, message.MsgType_ReplyErr, message.MsgType_ReplyErrConnNotExist, message.MsgType_ReplyErrLenLimit, message.MsgType_ReplyErrCheckSum:
				s.recvLock.Lock()
				ch, ok := s.recvChan[msg.Id]
				if ok {
					ch <- msg
					delete(s.recvChan, msg.Id)
				}
				s.recvLock.Unlock()
			default:
				s.handleOnCustomMessages(nodeNet.NewContext(c, msg, true))
			}
			continue
		}
		// 转发消息：优先转发到直连连接
		if dstConn, exist := s.ConnManager.Get(msg.DestId); exist {
			if _, err = dstConn.WriteMsg(msg); err == nil {
				continue
			}
		}
		via, ok := s.GetRoute(msg.DestId)
		if ok {
			if dstConn, exist := s.ConnManager.Get(via); exist {
				if _, err = dstConn.WriteMsg(msg); err == nil {
					continue
				}
			}
		}
		s.handleOnNoRouteMessage(nodeNet.NewContext(c, msg, true))
		continue
	}
}

func (s *Server) handleOnConnections(conn iface.Conn) {
	for _, callback := range s.OnConnections {
		callback(conn)
	}
}

func (s *Server) handleOnMessages(ctx iface.Context) {
	for _, callback := range s.OnMessages {
		callback(ctx)
		if !ctx.Next() {
			return
		}
	}
}

func (s *Server) handleOnCustomMessages(ctx iface.Context) {
	for _, callback := range s.OnCustomMessages {
		callback(ctx)
		if !ctx.Next() {
			return
		}
	}
}

func (s *Server) handleOnClosed(conn iface.Conn, err error) {
	for _, callback := range s.OnCloseds {
		callback(conn, err)
	}
}

func (s *Server) handleOnNoRouteMessage(ctx iface.Context) {
	if len(s.OnNoRouteMessage) == 0 {
		_ = ctx.CustomReply(message.MsgType_ReplyErrConnNotExist, nil)
		return
	}
	for _, callback := range s.OnNoRouteMessage {
		callback(ctx)
		if !ctx.Next() {
			return
		}
	}
}

func (s *Server) AddOnConnection(callback func(conn iface.Conn)) {
	s.OnConnections = append(s.OnConnections, callback)
}

func (s *Server) AddOnMessage(callback func(conn iface.Context)) {
	s.OnMessages = append(s.OnMessages, callback)
}

func (s *Server) AddOnCustomMessage(callback func(conn iface.Context)) {
	s.OnCustomMessages = append(s.OnCustomMessages, callback)
}

func (s *Server) AddOnClosed(callback func(conn iface.Conn, err error)) {
	s.OnCloseds = append(s.OnCloseds, callback)
}

func (s *Server) AddOnNoRouteMessage(callback func(conn iface.Context)) {
	s.OnNoRouteMessage = append(s.OnNoRouteMessage, callback)
}

// Bridge 桥接一个域,使用一个客户端连接到其他节点并绑定到当前节点形成一个大的域
func (s *Server) Bridge(conn net.Conn, remoteAuthKey []byte, timeout time.Duration) (rid uint32, err error) {
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	if err = defaultBasicReq.Send(conn, s.Identity.Id, remoteAuthKey, NodeType_Bridge); err != nil {
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
		return rid, errors.New("node ID cannot be the same as the server node ID")
	}
	c := nodeNet.NewConn(s.Identity.Id, rid, conn, s.recvChan, &s.recvLock, &s.counter, s.ReaderBufSize, s.WriterBufSize, s.WriterQueueSize, s.MaxMsgLen, uint8(NodeType_Bridge))
	if !s.ConnManager.Add(rid, c) {
		return rid, errors.New("node already exists")
	}
	go s.startConn(c)
	return rid, nil
}

// Request 请求
func (s *Server) Request(ctx context.Context, dst uint32, data []byte) ([]byte, error) {
	conn, exist := s.GetConn(dst)
	if exist {
		return conn.Request(ctx, data)
	}
	via, exist := s.GetRoute(dst)
	if exist {
		if conn, exist = s.GetConn(via); exist {
			return conn.Forward(ctx, dst, data)
		}
	}
	return nil, nodeNet.DEFAULT_ErrConnNotExist
}

func (s *Server) WriteTo(dst uint32, data []byte) (int, error) {
	conn, exist := s.GetConn(dst)
	if exist {
		return conn.Write(data)
	}
	via, exist := s.GetRoute(dst)
	if exist {
		if conn, exist = s.GetConn(via); exist {
			return conn.WriteTo(dst, data)
		}
	}
	return 0, nodeNet.DEFAULT_ErrConnNotExist
}

// GetConn 获取直连连接
func (s *Server) GetConn(id uint32) (iface.Conn, bool) {
	return s.ConnManager.Get(id)
}

// GetAllConn 获取全部直连连接
func (s *Server) GetAllConn() []iface.Conn {
	return s.ConnManager.GetAll()
}

func (s *Server) Id() uint32 {
	return s.Identity.Id
}

func (s *Server) Close() error {
	return s.listen.Close()
}

func BytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

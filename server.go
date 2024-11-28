package node

import (
	"errors"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	nodeNet "github.com/Li-giegie/node/net"
	"net"
	"sync"
	"time"
)

// NewServer 创建一个Server类型的节点,identity为节点标识必须设置，config为nil时使用默认配置
func NewServer(identity *Identity, c *Config) iface.Server {
	srv := new(Server)
	srv.id = identity.Id
	srv.authHashKey = hash(identity.Key)
	srv.authTimeout = identity.Timeout
	srv.connManage = nodeNet.NewConnManager()
	srv.recvChan = make(map[uint32]chan *message.Message)
	srv.closeChan = make(chan struct{}, 1)
	srv.Config = c
	if srv.Config == nil {
		srv.Config = defaultConfig
	}
	return srv
}

type Server struct {
	id          uint32
	authHashKey []byte
	authTimeout time.Duration
	recvChan    map[uint32]chan *message.Message
	recvLock    sync.Mutex
	counter     uint32
	closeChan   chan struct{}
	connManage  iface.ConnManager
	connectionEvent
	*Config
}

func (s *Server) Serve(l net.Listener) error {
	errChan := make(chan error, 1)
	go func() {
		for {
			if s.MaxConns > 0 && s.connManage.Len() > s.MaxConns {
				time.Sleep(s.MaxConnSleep)
				continue
			}
			conn, err := l.Accept()
			if err != nil {
				errChan <- err
				return
			}
			go s.handleAuthenticate(conn)
		}
	}()
	select {
	case err := <-errChan:
		_ = l.Close()
		return err
	case <-s.closeChan:
		_ = l.Close()
		return nil
	}
}

func (s *Server) handleAuthenticate(conn net.Conn) {
	src, dst, key, nt, err := defaultBasicReq.Receive(conn, s.authTimeout)
	if err != nil {
		_ = conn.Close()
		return
	}
	if src == s.id {
		_ = defaultBasicResp.Send(conn, false, "error: local id invalid")
		_ = conn.Close()
		return
	}
	if dst != s.id {
		_ = defaultBasicResp.Send(conn, false, "error: remote id invalid")
		_ = conn.Close()
		return
	}
	if !bytesEqual(s.authHashKey, key) {
		_ = defaultBasicResp.Send(conn, false, "error: key invalid")
		_ = conn.Close()
		return
	}
	c := nodeNet.NewConn(s.id, src, conn, s.recvChan, &s.recvLock, &s.counter, s.ReaderBufSize, s.WriterBufSize, s.WriterQueueSize, s.MaxMsgLen, uint8(nt))
	if !s.connManage.Add(src, c) {
		_ = defaultBasicResp.Send(conn, false, "error: id already exists")
		_ = conn.Close()
		return
	}
	if err = defaultBasicResp.Send(conn, true, ""); err != nil {
		_ = conn.Close()
		s.connManage.Remove(c.RemoteId())
		return
	}
	s.startConn(c)
}

func (s *Server) startConn(c *nodeNet.Connect) {
	s.onConnect(c)
	for {
		msg, err := c.ReadMsg()
		if err != nil {
			_ = c.Close()
			s.connManage.Remove(c.RemoteId())
			s.onClose(c, err)
			return
		}
		msg.Hop++
		// 当前节点消息
		if msg.DestId == s.id {
			switch msg.Type {
			case message.MsgType_Send:
				s.onMessage(nodeNet.NewContext(c, msg, true))
			case message.MsgType_Reply, message.MsgType_ReplyErr:
				s.recvLock.Lock()
				ch, ok := s.recvChan[msg.Id]
				if ok {
					ch <- msg
					delete(s.recvChan, msg.Id)
				}
				s.recvLock.Unlock()
			default:
				s.onCustomMessage(nodeNet.NewContext(c, msg, true))
			}
			continue
		}
		// 转发消息：优先转发到直连连接
		if dstConn, exist := s.connManage.Get(msg.DestId); exist {
			if _, err = dstConn.WriteMsg(msg); err == nil {
				continue
			}
		}
		s.onForwardMessage(nodeNet.NewContext(c, msg, true))
		continue
	}
}

func (s *Server) Bridge(conn net.Conn, remoteId uint32, remoteAuthKey []byte, timeout time.Duration) (err error) {
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	if remoteId == s.id {
		return errors.New("error: remote id id invalid")
	}
	if _, ok := s.GetConn(remoteId); ok {
		return errors.New("error: remote id already exists")
	}
	if err = defaultBasicReq.Send(conn, s.id, remoteId, remoteAuthKey, NodeType_Bridge); err != nil {
		return err
	}
	permit, msg, err := defaultBasicResp.Receive(conn, timeout)
	if err != nil {
		return err
	}
	if !permit {
		return errors.New(msg)
	}
	c := nodeNet.NewConn(s.id, remoteId, conn, s.recvChan, &s.recvLock, &s.counter, s.ReaderBufSize, s.WriterBufSize, s.WriterQueueSize, s.MaxMsgLen, uint8(NodeType_Bridge))
	if !s.connManage.Add(remoteId, c) {
		return errors.New("node already exists")
	}
	go s.startConn(c)
	return nil
}

// GetConn 获取直连连接
func (s *Server) GetConn(id uint32) (iface.Conn, bool) {
	return s.connManage.Get(id)
}

// GetAllConn 获取全部直连连接
func (s *Server) GetAllConn() []iface.Conn {
	return s.connManage.GetAll()
}

func (s *Server) Id() uint32 {
	return s.id
}

func (s *Server) Close() {
	for _, conn := range s.connManage.GetAll() {
		_ = conn.Close()
	}
	s.closeChan <- struct{}{}
}

func bytesEqual(a, b []byte) bool {
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

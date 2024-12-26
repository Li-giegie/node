package node

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	nodeNet "github.com/Li-giegie/node/net"
	"github.com/Li-giegie/node/router"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// NewServer 创建一个Server类型的节点,identity为节点标识必须设置，config为nil时使用默认配置
func NewServer(identity *Identity, c ...*Config) iface.Server {
	srv := new(Server)
	srv.init(identity, c...)
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
	*router.Router
	*nodeNet.ConnectionLifecycle
	*nodeNet.ConnManager
	*Config
}

func (s *Server) init(identity *Identity, c ...*Config) {
	s.id = identity.Id
	s.authTimeout = identity.AuthTimeout
	s.recvChan = make(map[uint32]chan *message.Message)
	s.closeChan = make(chan struct{}, 1)
	s.authHashKey = nodeNet.BaseAuthHash(identity.Key)
	s.ConnManager = nodeNet.NewConnManager()
	s.ConnectionLifecycle = nodeNet.NewConnectionLifecycle()
	s.Router = router.NewRouter()
	if n := len(c); n > 0 {
		if n != 1 {
			panic("config accepts only one parameter")
		}
		s.Config = c[0]
	} else {
		s.Config = DefaultConfig
	}
	if s.Config.MaxConnSleep == 0 && s.Config.MaxConns > 0 {
		panic("MaxConnSleep Must be greater than 0")
	}
}

func (s *Server) ListenAndServe(address string, conf ...*tls.Config) (err error) {
	var listen net.Listener
	network, addr := parseAddr(address)
	if n := len(conf); n > 0 {
		if n != 1 {
			panic("config accepts only one parameter")
		}
		listen, err = tls.Listen(network, addr, conf[0])
	} else {
		listen, err = net.Listen(network, addr)
	}
	if err != nil {
		return err
	}
	return s.Serve(listen)
}

func (s *Server) Serve(l net.Listener) error {
	errChan := make(chan error, 1)
	go func() {
		for {
			if s.MaxConns > 0 && s.ConnManager.Len() > s.MaxConns {
				time.Sleep(s.MaxConnSleep)
				continue
			}
			conn, err := l.Accept()
			if err != nil {
				errChan <- err
				return
			}
			go func() {
				if !s.CallOnAccept(conn) {
					_ = conn.Close()
					return
				}
				s.handleAuthenticate(conn)
			}()
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
	src, dst, key, err := nodeNet.DefaultBasicReq.Receive(conn, s.authTimeout)
	if err != nil {
		_ = conn.Close()
		return
	}
	if src == s.id {
		_ = nodeNet.DefaultBasicResp.Send(conn, false, "error: local id invalid")
		_ = conn.Close()
		return
	}
	if dst != s.id {
		_ = nodeNet.DefaultBasicResp.Send(conn, false, "error: remote id invalid")
		_ = conn.Close()
		return
	}
	if !bytesEqual(s.authHashKey, key) {
		_ = nodeNet.DefaultBasicResp.Send(conn, false, "error: key invalid")
		_ = conn.Close()
		return
	}
	c := nodeNet.NewConn(s.id, src, conn, s.recvChan, &s.recvLock, &s.counter, s.ReaderBufSize, s.WriterBufSize, s.WriterQueueSize, s.MaxMsgLen)
	if !s.AddConn(src, c) {
		_ = nodeNet.DefaultBasicResp.Send(conn, false, "error: id already exists")
		_ = conn.Close()
		return
	}
	if err = nodeNet.DefaultBasicResp.Send(conn, true, ""); err != nil {
		_ = conn.Close()
		s.RemoveConn(c.RemoteId())
		return
	}
	s.startConn(c)
}

func (s *Server) startConn(c *nodeNet.Conn) {
	s.CallOnConnect(c)
	for {
		msg, err := c.ReadMessage()
		if err != nil {
			_ = c.Close()
			s.RemoveConn(c.RemoteId())
			s.CallOnClose(c, err)
			return
		}
		msg.Hop++
		// 当前节点消息
		if msg.DestId != s.id {
			// 转发消息：优先转发到直连连接
			if dstConn, exist := s.GetConn(msg.DestId); exist {
				if err = dstConn.SendMessage(msg); err == nil {
					continue
				}
			}
			route, ok := s.GetRoute(msg.DestId)
			if !ok {
				_ = nodeNet.NewContext(c, msg).Response(message.StateCode_NodeNotExist, nil)
				continue
			}
			conn, ok := s.GetConn(route.Via)
			if !ok {
				_ = nodeNet.NewContext(c, msg).Response(message.StateCode_NodeNotExist, nil)
				s.RemoveRoute(route.Dst, route.UnixNano)
				continue
			}
			_ = conn.SendMessage(msg)
			continue
		}
		if msg.Type == message.MsgType_Reply {
			s.recvLock.Lock()
			ch, ok := s.recvChan[msg.Id]
			if ok {
				ch <- msg
				delete(s.recvChan, msg.Id)
			}
			s.recvLock.Unlock()
		} else {
			s.CallOnMessage(nodeNet.NewContext(c, msg))
		}
	}
}

func (s *Server) RequestTo(ctx context.Context, dst uint32, data []byte) ([]byte, int16, error) {
	return s.RequestTypeTo(ctx, message.MsgType_Default, dst, data)
}

func (s *Server) RequestTypeTo(ctx context.Context, typ uint8, dst uint32, data []byte) ([]byte, int16, error) {
	return s.RequestMessage(ctx, &message.Message{
		Type:   typ,
		Id:     atomic.AddUint32(&s.counter, 1),
		SrcId:  s.id,
		DestId: dst,
		Data:   data,
	})
}

func (s *Server) RequestMessage(ctx context.Context, msg *message.Message) ([]byte, int16, error) {
	conn, ok := s.GetConn(msg.DestId)
	if ok {
		return conn.RequestMessage(ctx, msg)
	}
	route, ok := s.Router.GetRoute(msg.DestId)
	if ok {
		if conn, ok = s.GetConn(route.Via); ok {
			return conn.RequestMessage(ctx, msg)
		}
	}
	return nil, message.StateCode_NodeNotExist, nil
}

func (s *Server) SendTo(dst uint32, data []byte) error {
	return s.SendTypeTo(message.MsgType_Default, dst, data)
}

func (s *Server) SendTypeTo(typ uint8, dst uint32, data []byte) error {
	return s.SendMessage(&message.Message{
		Type:   typ,
		Id:     atomic.AddUint32(&s.counter, 1),
		SrcId:  s.id,
		DestId: dst,
		Data:   data,
	})
}

func (s *Server) SendMessage(msg *message.Message) error {
	conn, ok := s.GetConn(msg.DestId)
	if ok {
		return conn.SendMessage(msg)
	}
	route, ok := s.Router.GetRoute(msg.DestId)
	if ok {
		if conn, ok = s.GetConn(route.Via); ok {
			return conn.SendMessage(msg)
		}
	}
	return nodeNet.ErrNodeNotExist
}

func (s *Server) Bridge(conn net.Conn, remoteId uint32, remoteAuthKey []byte) (err error) {
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
	if err = nodeNet.DefaultBasicReq.Send(conn, s.id, remoteId, remoteAuthKey); err != nil {
		return err
	}
	permit, msg, err := nodeNet.DefaultBasicResp.Receive(conn, s.authTimeout)
	if err != nil {
		return err
	}
	if !permit {
		return errors.New(msg)
	}
	c := nodeNet.NewConn(s.id, remoteId, conn, s.recvChan, &s.recvLock, &s.counter, s.ReaderBufSize, s.WriterBufSize, s.WriterQueueSize, s.MaxMsgLen)
	if !s.AddConn(remoteId, c) {
		return errors.New("node already exists")
	}
	go s.startConn(c)
	return nil
}

func (s *Server) GetRouter() iface.Router {
	return s.Router
}

func (s *Server) Id() uint32 {
	return s.id
}

func (s *Server) Close() {
	for _, conn := range s.GetAllConn() {
		_ = conn.Close()
	}
	s.closeChan <- struct{}{}
}

package impl_server

import (
	"context"
	"crypto/tls"
	"github.com/Li-giegie/node/internal"
	"github.com/Li-giegie/node/internal/connmanager/impl_connmanager"
	"github.com/Li-giegie/node/internal/eventhandlerregistry/impl_eventhandlerregistry"
	"github.com/Li-giegie/node/pkg/common"
	"github.com/Li-giegie/node/pkg/conn/impl_conn"
	"github.com/Li-giegie/node/pkg/ctx/impl_context"
	"github.com/Li-giegie/node/pkg/errors/impl_errors"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/router"
	"github.com/Li-giegie/node/pkg/router/impl_router"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// NewServer 创建一个Server类型的节点,identity为节点标识必须设置，config为nil时使用默认配置
func NewServer(identity *common.Identity, c ...*common.Config) *Server {
	s := new(Server)
	s.id = identity.Id
	s.authTimeout = identity.AuthTimeout
	s.recvChan = make(map[uint32]chan *message.Message)
	s.stopCtx, s.cancel = context.WithCancel(context.Background())
	s.authHashKey = internal.Hash(identity.Key)
	s.ConnManager = impl_connmanager.NewConnManager()
	s.EventHandlerRegistry = impl_eventhandlerregistry.NewEventHandlerRegistry()
	s.Router = impl_router.NewRouter()
	if n := len(c); n > 0 {
		if n != 1 {
			panic(impl_errors.MultipleConfigErr)
		}
		s.Config = c[0]
	} else {
		s.Config = common.DefaultConfig
	}
	return s
}

type Server struct {
	id          uint32
	authHashKey []byte
	authTimeout time.Duration
	recvChan    map[uint32]chan *message.Message
	recvLock    sync.Mutex
	counter     uint32
	stopCtx     context.Context
	cancel      context.CancelFunc
	*impl_router.Router
	*impl_eventhandlerregistry.EventHandlerRegistry
	*impl_connmanager.ConnManager
	*common.Config
}

func (s *Server) ListenAndServe(address string, conf ...*tls.Config) (err error) {
	var listen net.Listener
	network, addr := internal.ParseAddr(address)
	if n := len(conf); n > 0 {
		if n != 1 {
			return impl_errors.MultipleConfigErr
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
	if s.Config.MaxConnSleep == 0 && s.Config.MaxConns > 0 {
		return impl_errors.ConfigMaxConnSleepErr
	}
	s.startHeartbeatCheck()
	errChan := make(chan error, 1)
	go func() {
		for {
			if s.MaxConns > 0 && s.ConnManager.LenConn() > s.MaxConns {
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
	case <-s.stopCtx.Done():
		_ = l.Close()
		return nil
	}
}

func (s *Server) handleAuthenticate(conn net.Conn) {
	src, dst, key, err := internal.DefaultBasicReq.Receive(conn, s.authTimeout)
	if err != nil {
		_ = conn.Close()
		return
	}
	if src == s.id {
		_ = internal.DefaultBasicResp.Send(conn, false, "error: local id invalid")
		_ = conn.Close()
		return
	}
	if dst != s.id {
		_ = internal.DefaultBasicResp.Send(conn, false, "error: remote id invalid")
		_ = conn.Close()
		return
	}
	if !internal.BytesEqual(s.authHashKey, key) {
		_ = internal.DefaultBasicResp.Send(conn, false, "error: key invalid")
		_ = conn.Close()
		return
	}
	c := impl_conn.NewConn(s.id, src, conn, s.recvChan, &s.recvLock, &s.counter, s.ReaderBufSize, s.WriterBufSize, s.WriterQueueSize, s.MaxMsgLen)
	if !s.AddConn(src, c) {
		_ = internal.DefaultBasicResp.Send(conn, false, "error: id already exists")
		_ = conn.Close()
		return
	}
	if err = internal.DefaultBasicResp.Send(conn, true, ""); err != nil {
		_ = conn.Close()
		s.RemoveConn(c.RemoteId())
		return
	}
	s.startConn(c)
}

func (s *Server) startConn(c *impl_conn.Conn) {
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
		if msg.DestId != s.id {
			// 本地存在
			if dstConn, exist := s.GetConn(msg.DestId); exist {
				_ = dstConn.SendMessage(msg)
				continue
			}
			// 查路由存在
			if route, ok := s.GetRoute(msg.DestId); ok {
				if conn, ok := s.GetConn(route.Via); ok {
					_ = conn.SendMessage(msg)
					continue
				}
				// 路由表更新不及时
				s.RemoveRoute(route.Dst, route.UnixNano)
			}
			_ = impl_context.NewContext(c, msg).Response(message.StateCode_NodeNotExist, nil)
			continue
		}
		switch msg.Type {
		case message.MsgType_KeepaliveASK:
			_ = c.SendType(message.MsgType_KeepaliveACK, nil)
		case message.MsgType_KeepaliveACK:
		case message.MsgType_Reply:
			s.recvLock.Lock()
			ch, ok := s.recvChan[msg.Id]
			if ok {
				ch <- msg
				delete(s.recvChan, msg.Id)
			}
			s.recvLock.Unlock()
		default:
			s.CallOnMessage(impl_context.NewContext(c, msg))
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
	return impl_errors.ErrNodeNotExist
}

func (s *Server) Bridge(conn net.Conn, remoteId uint32, remoteAuthKey []byte) (err error) {
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	if remoteId == s.id {
		return impl_errors.BridgeRemoteIdExistErr
	}
	if _, ok := s.GetConn(remoteId); ok {
		return impl_errors.BridgeRemoteIdExistErr
	}
	if err = internal.DefaultBasicReq.Send(conn, s.id, remoteId, remoteAuthKey); err != nil {
		return err
	}
	permit, msg, err := internal.DefaultBasicResp.Receive(conn, s.authTimeout)
	if err != nil {
		return err
	}
	if !permit {
		return impl_errors.NodeError(msg)
	}
	c := impl_conn.NewConn(s.id, remoteId, conn, s.recvChan, &s.recvLock, &s.counter, s.ReaderBufSize, s.WriterBufSize, s.WriterQueueSize, s.MaxMsgLen)
	if !s.AddConn(remoteId, c) {
		return impl_errors.BridgeRemoteIdExistErr
	}
	go s.startConn(c)
	return nil
}

func (s *Server) GetRouter() router.Router {
	return s.Router
}

func (s *Server) Id() uint32 {
	return s.id
}

func (s *Server) Close() {
	for _, conn := range s.GetAllConn() {
		_ = conn.Close()
	}
	s.cancel()
}

func (s *Server) SetKeepalive(interval, timeout, timeoutClose time.Duration) {
	s.KeepaliveInterval = interval
	s.KeepaliveTimeout = timeout
	s.KeepaliveTimeoutClose = timeoutClose
	if !s.Keepalive {
		s.Keepalive = true
		s.startHeartbeatCheck()
	}
}

func (s *Server) startHeartbeatCheck() {
	if !s.Keepalive {
		return
	}
	go func() {
		tick := time.NewTicker(s.KeepaliveInterval)
		defer tick.Stop()
		go func() {
			<-s.stopCtx.Done()
			tick.Stop()
		}()
		var err error
		var diff int64
		for t := range tick.C {
			for _, conn := range s.GetAllConn() {
				diff = t.UnixNano() - int64(conn.Activate())
				if diff > int64(s.KeepaliveTimeoutClose) {
					_ = conn.Close()
				} else if diff > int64(s.KeepaliveTimeout) {
					if err = conn.SendType(message.MsgType_KeepaliveASK, nil); err != nil {
						_ = conn.Close()
					}
				}
			}
		}
	}()
	return
}

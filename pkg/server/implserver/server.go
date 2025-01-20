package implserver

import (
	"context"
	"crypto/tls"
	"github.com/Li-giegie/node/internal"
	"github.com/Li-giegie/node/internal/connmanager"
	"github.com/Li-giegie/node/internal/handlermanager"
	"github.com/Li-giegie/node/internal/routemanager"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/conn/implconn"
	"github.com/Li-giegie/node/pkg/errors"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/router"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Server struct {
	// 节点Id
	Id uint32
	// 节点认证key
	AuthKey []byte
	// 认证超时
	AuthTimeout time.Duration
	// 大于0时启用，收发消息最大长度，最大值0xffffffff
	MaxMsgLen uint32
	// 大于1时启用，并发请求或发送时，发出的消息不会被立即发出，而是会进入队列，直至队列缓冲区满，或者最后一个goroutine时才会将消息发出，如果消息要以最快的方式发出，那么请不要进入队列
	WriterQueueSize int
	// 读缓存区大小
	ReaderBufSize int
	// 大于64时启用，从队列读取后进入缓冲区，缓冲区大小
	WriterBufSize int
	// 大于0启用，最大连接数
	MaxConnections int
	// 超过最大连接休眠时长，MaxConns>0时有效
	SleepOnMaxConnections time.Duration
	// 连接保活检查时间间隔 > 0启用
	KeepaliveInterval time.Duration
	// 连接保活超时时间 > 0启用
	KeepaliveTimeout time.Duration
	// 连接保活最大超时次数
	KeepaliveTimeoutClose time.Duration
	MaxHop                uint8
	internalField
}

type internalField struct {
	hashKey   []byte
	idCounter uint32
	recvChan  map[uint32]chan *message.Message
	recvLock  sync.Mutex
	stopCtx   context.Context
	cancel    context.CancelFunc
	routemanager.Router
	connmanager.ConnManager
	handlemanager.HandlerManager
}

func (s *Server) ListenAndServe(address string, conf ...*tls.Config) (err error) {
	var listen net.Listener
	network, addr := internal.ParseAddr(address)
	if n := len(conf); n > 0 {
		if n != 1 {
			return errors.MultipleConfigErr
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
	defer l.Close()
	s.stopCtx, s.cancel = context.WithCancel(context.Background())
	s.recvChan = make(map[uint32]chan *message.Message)
	s.hashKey = internal.Hash(s.AuthKey)
	s.startHeartbeatCheck()
	errChan := make(chan error, 1)
	go func() {
		for {
			if s.MaxConnections > 0 && s.LenConn() > s.MaxConnections {
				time.Sleep(s.SleepOnMaxConnections)
				continue
			}
			native, err := l.Accept()
			if err != nil {
				errChan <- err
				return
			}
			go func() {
				if !s.CallOnAccept(native) {
					_ = native.Close()
					return
				}
				s.handleAuthenticate(native)
			}()
		}
	}()
	select {
	case err := <-errChan:
		return err
	case <-s.stopCtx.Done():
		return nil
	}
}

func (s *Server) handleAuthenticate(native net.Conn) {
	srcType, src, dst, key, err := internal.DefaultBasicReq.Receive(native, s.AuthTimeout)
	if err != nil {
		_ = native.Close()
		return
	}
	if src == s.Id {
		_ = internal.DefaultBasicResp.Send(native, 0, false, "error: local id invalid")
		_ = native.Close()
		return
	}
	if dst != s.Id {
		_ = internal.DefaultBasicResp.Send(native, 0, false, "error: remote id invalid")
		_ = native.Close()
		return
	}
	if !internal.BytesEqual(s.hashKey, key) {
		_ = internal.DefaultBasicResp.Send(native, 0, false, "error: key invalid")
		_ = native.Close()
		return
	}
	c := implconn.NewConn(srcType, s.Id, src, native, s.recvChan, &s.recvLock, &s.idCounter, s.ReaderBufSize, s.WriterBufSize, s.WriterQueueSize, s.MaxMsgLen)
	if !s.AddConn(c) {
		_ = internal.DefaultBasicResp.Send(native, 0, false, "error: id already exists")
		_ = native.Close()
		return
	}
	if err = internal.DefaultBasicResp.Send(native, conn.NodeTypeServer, true, ""); err != nil {
		_ = native.Close()
		s.RemoveConn(c.RemoteId())
		return
	}
	s.handleConn(c)
}

func (s *Server) handleConn(c *implconn.Conn) {
	s.CallOnConnect(c)
	for {
		msg, err := c.ReadMessage()
		if err != nil {
			_ = c.Close()
			s.RemoveConn(c.RemoteId())
			s.CallOnClose(c, err)
			return
		}
		if msg.Hop >= 254 || msg.Hop >= s.MaxHop && s.MaxHop > 0 {
			continue
		}
		msg.Hop++
		if msg.DestId != s.Id {
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
				s.RemoveRoute(route.Dst)
			}
			r := internal.ResponseWriter{
				Conn:     c,
				MsgId:    msg.Id,
				MsgDstId: msg.SrcId,
			}
			_ = r.Response(message.StateCode_NodeNotExist, nil)
			continue
		}
		switch msg.Type {
		case message.MsgType_KeepaliveASK:
			_ = c.SendType(message.MsgType_KeepaliveACK, nil)
		case message.MsgType_KeepaliveACK:
		case message.MsgType_Response:
			s.recvLock.Lock()
			ch, ok := s.recvChan[msg.Id]
			if ok {
				ch <- msg
				delete(s.recvChan, msg.Id)
			}
			s.recvLock.Unlock()
		default:
			s.CallOnMessage(&internal.ResponseWriter{Conn: c, MsgId: msg.Id, MsgDstId: msg.SrcId}, msg)
		}
	}
}

func (s *Server) RequestTo(ctx context.Context, dst uint32, data []byte) ([]byte, int16, error) {
	return s.RequestTypeTo(ctx, message.MsgType_Default, dst, data)
}

func (s *Server) RequestTypeTo(ctx context.Context, typ uint8, dst uint32, data []byte) ([]byte, int16, error) {
	return s.RequestMessage(ctx, &message.Message{
		Type:   typ,
		Id:     atomic.AddUint32(&s.idCounter, 1),
		SrcId:  s.Id,
		DestId: dst,
		Data:   data,
	})
}

func (s *Server) RequestMessage(ctx context.Context, msg *message.Message) ([]byte, int16, error) {
	conn, ok := s.GetConn(msg.DestId)
	if ok {
		return conn.RequestMessage(ctx, msg)
	}
	route, ok := s.GetRoute(msg.DestId)
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
		Id:     atomic.AddUint32(&s.idCounter, 1),
		SrcId:  s.Id,
		DestId: dst,
		Data:   data,
	})
}

func (s *Server) SendMessage(msg *message.Message) error {
	conn, ok := s.GetConn(msg.DestId)
	if ok {
		return conn.SendMessage(msg)
	}
	route, ok := s.GetRoute(msg.DestId)
	if ok {
		if conn, ok = s.GetConn(route.Via); ok {
			return conn.SendMessage(msg)
		}
	}
	return errors.ErrNodeNotExist
}

func (s *Server) Bridge(native net.Conn, remoteId uint32, remoteAuthKey []byte) (err error) {
	defer func() {
		if err != nil {
			_ = native.Close()
		}
	}()
	if remoteId == s.Id {
		return errors.BridgeRemoteIdExistErr
	}
	if _, ok := s.GetConn(remoteId); ok {
		return errors.BridgeRemoteIdExistErr
	}
	if err = internal.DefaultBasicReq.Send(native, conn.NodeTypeServer, s.Id, remoteId, remoteAuthKey); err != nil {
		return err
	}
	dstType, permit, msg, err := internal.DefaultBasicResp.Receive(native, s.AuthTimeout)
	if err != nil {
		return err
	}
	if !permit {
		return errors.Error(msg)
	}
	c := implconn.NewConn(dstType, s.Id, remoteId, native, s.recvChan, &s.recvLock, &s.idCounter, s.ReaderBufSize, s.WriterBufSize, s.WriterQueueSize, s.MaxMsgLen)
	if !s.AddConn(c) {
		return errors.BridgeRemoteIdExistErr
	}
	go s.handleConn(c)
	return nil
}

func (s *Server) NodeId() uint32 {
	return s.Id
}

func (s *Server) Close() {
	if s.cancel == nil {
		return
	}
	for _, conn := range s.GetAllConn() {
		_ = conn.Close()
	}
	s.cancel()
}

func (s *Server) startHeartbeatCheck() {
	if s.KeepaliveInterval <= 0 || s.KeepaliveTimeout <= 0 {
		return
	}
	go func() {
		if s.KeepaliveTimeoutClose < s.KeepaliveTimeout {
			s.KeepaliveTimeoutClose = s.KeepaliveTimeout
		}
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

func (s *Server) GetRouter() router.Router {
	return &s.Router
}

func (s *Server) CreateMessageId() uint32 {
	return atomic.AddUint32(&s.idCounter, 1)
}

func (s *Server) CreateMessage(typ uint8, src uint32, dst uint32, data []byte) *message.Message {
	return &message.Message{
		Type:   typ,
		Id:     atomic.AddUint32(&s.idCounter, 1),
		SrcId:  src,
		DestId: dst,
		Data:   data,
	}
}

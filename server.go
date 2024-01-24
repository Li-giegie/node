package node

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"log"
	"net"
	"time"
)

type IServer interface {
	init() (addr *net.TCPAddr, err error)
	HandleFunc(api uint32, handler HandlerFunc)
	ListenAndServer(debug ...bool) error
	newConnect(conn *net.TCPConn)
	authConnect(msg *authMsg) ([]byte, error)
	process(ctx *srvConnCtx) error
	GetConnect(id uint64) (ISrvConn, bool) //获取一个连接
	GetConnList() []ISrvConn
	Shutdown()
}

type Option func(server *Server) error

type Server struct {
	id         uint64
	state      bool
	addr       string
	maxConnNum int
	gPool      *ants.Pool
	listen     *net.TCPListener
	iHandler
	iConnectList
	iRegisterHandle
	AuthenticationFunc
	*ServerTimeParameters
	*ServerGoroutineParameters
}

type ServerTimeParameters struct {
	//最大连接空闲时间
	MaxConnectionIdle time.Duration
	//检查连接是否有效间隔时间
	CheckInterval time.Duration
}

type ServerGoroutineParameters struct {
	//开启的Goroutine数量
	GoroutineNum int
	//扩容Goroutine最大数量，MaxGoroutine > GoroutineNum 有效
	MaxGoroutine int
}

func NewServer(addr string, opt ...Option) IServer {
	srv := new(Server)
	srv.ServerTimeParameters = new(ServerTimeParameters)
	srv.ServerGoroutineParameters = new(ServerGoroutineParameters)
	srv.CheckInterval = DEFAULT_CheckInterval
	srv.MaxConnectionIdle = DEFAULT_KeepAlive
	srv.MaxGoroutine = DEFAULT_MAX_GOROUTINE
	srv.GoroutineNum = DEFAULT_MIN_GOROUTINE
	srv.addr = addr
	srv.id = DEFAULT_ServerID
	srv.maxConnNum = DEFAULT_MAXCONNNUM
	srv.iConnectList = newConnectList()
	srv.iHandler = newHandler()
	srv.iRegisterHandle = newRegisterHandle()
	for _, option := range opt {
		_ = option(srv)
	}
	return srv
}

func (s *Server) ServerId() uint64 {
	return s.id
}

func (s *Server) HandleFunc(api uint32, handler HandlerFunc) {
	s.AddHandle(api, handler)
}

func (s *Server) getMaxConnectionIdle() time.Duration {
	return s.ServerTimeParameters.MaxConnectionIdle
}

func (s *Server) init() (addr *net.TCPAddr, err error) {
	s.gPool, err = ants.NewPool(s.GoroutineNum)
	if err != nil {
		return
	}
	s.gPool.Tune(s.MaxGoroutine)
	return net.ResolveTCPAddr("tcp", s.addr)
}

func (s *Server) ListenAndServer(debug ...bool) error {
	addr, err := s.init()
	if err != nil {
		return err
	}
	if s.listen, err = net.ListenTCP("tcp", addr); err != nil {
		return err
	}
	s.state = true
	defer s.listen.Close()
	if err = s.gPool.Submit(s.checkConnect); err != nil {
		return err
	}
	if len(debug) > 0 && debug[0] {
		log.Printf("server [%d] listen: %s\n", s.id, s.addr)
	}
	for s.state {
		conn, err := s.listen.AcceptTCP()
		if err != nil {
			return err
		}
		log.Printf("[debug] listen conntion: %s\n", conn.RemoteAddr().String())
		err = s.gPool.Submit(func() {
			s.newConnect(conn)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) newConnect(conn *net.TCPConn) {
	if s.maxConnNum > 0 && s.Len() > s.maxConnNum {
		_ = write(conn, encodeErrReplyMsgData(ErrServerConnectOverFlow, nil))
		_ = conn.Close()
		log.Printf("[debug] close -1 connection: %s\n", conn.RemoteAddr().String())
		return
	}
	addr := conn.RemoteAddr().String()
	sessionId := randomU32()
	//发送一个session id uint32，接收消息时需要作为判断连接是否合法的依据之一，防止错误的连接建立造成意外情况
	_, err := conn.Write(uint32ToBytes(sessionId))
	if err != nil {
		_ = conn.Close()
		log.Printf("[debug] close -2 connection: %s\n", conn.RemoteAddr().String())
		return
	}
	amHeader, err := new(authHeader).unmarshal(conn)
	if err != nil {
		_ = write(conn, encodeErrReplyMsgData(err, nil))
		_ = conn.Close()
		log.Printf("[debug] close -3 connection: %s\n", conn.RemoteAddr().String())
		return
	}

	if sessionId != amHeader.sessionId || amHeader.version != Version || amHeader.dstId != s.id || amHeader.srcId == 0 {
		tmpBuf := encodeErrReplyMsgData(fmt.Errorf("%v -3 ", ErrInvalidConnect), nil)
		_ = write(conn, tmpBuf)
		_ = conn.Close()
		log.Printf("[debug] close -4 connection: %s\n", conn.RemoteAddr().String())
		return
	}
	am := new(authMsg)
	am.authHeader = amHeader
	err = am.unmarshal(conn)
	if err != nil {
		_ = write(conn, encodeErrReplyMsgData(err, nil))
		_ = conn.Close()
		log.Printf("[debug] close -5 connection: %s\n", conn.RemoteAddr().String())
		return
	}
	authData, err := s.authConnect(am)
	if err != nil {
		_ = write(conn, encodeErrReplyMsgData(err, authData))
		_ = conn.Close()
		log.Printf("[debug] close -6 connection: %s\n", conn.RemoteAddr().String())
		return
	}
	if err = write(conn, encodeErrReplyMsgData(nil, authData)); err != nil {
		_ = conn.Close()
		log.Printf("[debug] close -7 connection: %s\n", conn.RemoteAddr().String())
		return
	}
	log.Printf("[debug] success conntion: %d %s\n", am.srcId, addr)
	sConn := newSrvConn(am.srcId, conn, s)
	s.Add(sConn)
	err = s.gPool.Submit(sConn.start)
	if err != nil {
		log.Println("ants pool err: ", err)
	}
}

// authConnect authentication connect
func (s *Server) authConnect(auth *authMsg) ([]byte, error) {
	conn, ok := s.GetConnect(auth.srcId)
	if ok && conn.(*srvConn).Status {
		return nil, ErrAuthIdExist
	}
	if s.AuthenticationFunc == nil {
		return nil, nil
	}
	buf, err := s.AuthenticationFunc(auth.srcId, auth.data)
	if err != nil {
		return buf, err
	}
	return buf, nil
}

func (s *Server) process(ctx *srvConnCtx) error {
	return s.gPool.Submit(func() {
		switch ctx.msg.typ {
		case msgType_Send:
			switch ctx.msg.dstId {
			case s.id, 0: //本地处理
				hi, ok := s.GetHandle(ctx.msg.api)
				if ok {
					hi(newContext(ctx.msg, ctx.conn))
					return
				}
				conn, ok := s.QueryRegisterConn(ctx.msg.api)
				if !ok {
					ctx.msg.replyErr(msgType_ReplyErr, nil, ErrNoApi)
					_ = ctx.conn.writeMsg(ctx.msg)
					return
				}
				if conn == nil || !conn.Status {
					ctx.msg.replyErr(msgType_ReplyErr, nil, ErrNoApi)
					_ = ctx.conn.writeMsg(ctx.msg)
					return
				}
				ctx.msg.dstId = conn.Id
				if err := conn.writeMsg(ctx.msg); err != nil {
					log.Println("conn.process.localHandle.forward err: ", err)
				}
			default: //转发处理
				iConn, ok := s.GetConnect(ctx.msg.dstId)
				if !ok {
					ctx.msg.replyErr(msgType_ReplyErr, nil, ErrConnNotExist)
					_ = ctx.conn.writeMsg(ctx.msg)
					return
				}
				if err := iConn.(*srvConn).writeMsg(ctx.msg); err != nil {
					log.Println("conn.process.forward err: ", err)
				}
			}
		case msgType_Reply, msgType_ReplyErr, msgType_RegistrationReply, msgType_TickReply:
			switch ctx.msg.dstId {
			case s.id, 0:
				mChan, ok := ctx.conn.GetMsgChan(ctx.msg.id)
				if !ok {
					log.Println("No recipient drop message", ctx.msg.String())
					return
				}
				if mChan == nil {
					log.Println("message channel close drop message", ctx.msg.String())
					return
				}
				mChan <- ctx.msg
			default:
				conn, ok := s.GetConnect(ctx.msg.dstId)
				if ok {
					if err := conn.(*srvConn).writeMsg(ctx.msg); err != nil {
						log.Println("srvConn.process.forward err: ", err)
					}
					return
				}
				log.Println("drop reply message: ", ctx.msg.String())
			}
		case msgType_Registration:
			var ok bool
			apis := decodeRegistrationApiReq(ctx.msg.data)
			var badApis = make([]uint32, 0, len(apis))
			for _, api := range apis {
				if _, ok = s.GetHandle(api); ok {
					badApis = append(badApis, api)
				} else if conn, ok := s.QueryRegisterConn(api); ok {
					if conn != nil && conn.Status {
						badApis = append(badApis, api)
					}
				}
			}
			if len(badApis) > 0 {
				ctx.msg.reply(msgType_RegistrationReply, encodeRegistrationApiResp(ErrRegistrationApiExist, badApis))
				_ = ctx.conn.writeMsg(ctx.msg)
				return
			}
			s.AppendRegisterConn(ctx.conn, apis)
			ctx.conn.apis = apis
			ctx.msg.reply(msgType_RegistrationReply, encodeRegistrationApiResp(nil, nil))
			_ = ctx.conn.writeMsg(ctx.msg)
		case msgType_Tick:
			ctx.msg.reply(msgType_TickReply, nil)
			_ = ctx.conn.writeMsg(ctx.msg)
		}
	})
}

func (s *Server) ConnectEvent(cet connectEventType, arg ...interface{}) {
	switch cet {
	//手动关闭、检测超时关闭、读写超时关闭
	case connectEventType_Close, connectEventType_TimeOutClose, connectEventType_processClose:
		conn, ok := arg[0].(*srvConn)
		if ok && conn != nil {
			for _, api := range conn.apis {
				s.DeleteRegisterConn(api)
			}
			conn.close(arg[1].(bool))
		}
		s.Delete(conn.Id)
		log.Printf("close %s: id %d\n", connectEventMap[cet], conn.Id)
	default:
		log.Println("invalid event: ", cet)
	}
}

func (s *Server) GetConnect(id uint64) (ISrvConn, bool) {
	i, ok := s.Query(id)
	if !ok {
		return nil, false
	}
	return i, true
}

func (s *Server) GetConnList() []ISrvConn {
	key := s.Keys()
	list := make([]ISrvConn, 0, len(key))
	for _, id := range key {
		conn, ok := s.GetConnect(id)
		if ok {
			list = append(list, conn)
		}
	}
	return list
}

func (s *Server) checkConnect() {
	mci := int64(s.ServerTimeParameters.MaxConnectionIdle.Seconds())
	for s.state {
		time.Sleep(s.ServerTimeParameters.CheckInterval)
		keys := s.Keys()
		for _, key := range keys {
			conn, ok := s.GetConnect(key)
			if !ok {
				continue
			}
			if time.Now().Unix() > mci+conn.(*srvConn).activation {
				s.ConnectEvent(connectEventType_TimeOutClose, conn, true)
				log.Println("连接超时关闭：", key)
			}
		}
	}
}

func (s *Server) Shutdown() {
	_ = s.listen.Close()
	s.state = false
}

func WithSrvId(id uint64) Option {
	return func(srv *Server) error {
		srv.id = id
		return nil
	}
}

func WithSrvTimeParameters(t ServerTimeParameters) Option {
	return func(srv *Server) error {
		srv.ServerTimeParameters = &t
		return nil
	}
}

// WithSrvGoroutineParameters 设置goroutine相关
func WithSrvGoroutineParameters(g ServerGoroutineParameters) Option {
	return func(srv *Server) error {
		srv.ServerGoroutineParameters = &g
		return nil
	}
}

// WithSrvAuthentication Set authentication
func WithSrvAuthentication(authFunc AuthenticationFunc) Option {
	return func(srv *Server) error {
		srv.AuthenticationFunc = authFunc
		return nil
	}
}

// WithSrvMaxConnectNum <= 0 不限制连接数量
func WithSrvMaxConnectNum(maxNum int) Option {
	return func(srv *Server) error {
		srv.maxConnNum = maxNum
		return nil
	}
}

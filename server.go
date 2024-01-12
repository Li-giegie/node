package node

import (
	"fmt"
	utils "github.com/Li-giegie/go-utils"
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
	GetConnect(id uint64) (ISrvConn, bool)
	GetConnList() []ISrvConn
	Shutdown()
}

type Option func(server *Server) error

type Server struct {
	id                    uint64
	state                 bool
	addr                  string
	localHandle           *Handler
	registerHandleApiConn *utils.MapUint32
	maxGoroutine          int
	minGoroutine          int
	connKeepAliveTime     time.Duration
	maxConnNum            int
	connList              *utils.MapUint64
	gPool                 *ants.Pool
	AuthenticationFunc
}

func NewServer(addr string, opt ...Option) IServer {
	srv := new(Server)
	srv.addr = addr
	srv.state = true
	srv.id = DEFAULT_ServerID
	srv.maxGoroutine = DEFAULT_MAX_GOROUTINE
	srv.minGoroutine = DEFAULT_MIN_GOROUTINE
	srv.connKeepAliveTime = DEFAULT_KeepAlive
	srv.maxConnNum = DEFAULT_MAXCONNNUM
	srv.connList = utils.NewMapUint64()
	srv.localHandle = NewHandler()
	srv.registerHandleApiConn = utils.NewMapUint32()
	for _, option := range opt {
		_ = option(srv)
	}
	return srv
}

func (s *Server) ServerId() uint64 {
	return s.id
}

func (s *Server) HandleFunc(api uint32, handler HandlerFunc) {
	s.localHandle.Add(api, handler)
}

func (s *Server) init() (addr *net.TCPAddr, err error) {
	s.gPool, err = ants.NewPool(s.minGoroutine)
	if err != nil {
		return
	}
	s.gPool.Tune(s.maxGoroutine)
	return net.ResolveTCPAddr("tcp", s.addr)
}

func (s *Server) ListenAndServer(debug ...bool) error {
	addr, err := s.init()
	if err != nil {
		return err
	}
	listen, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	if err = s.gPool.Submit(s.checkUp); err != nil {
		return err
	}
	defer listen.Close()
	if len(debug) > 0 && debug[0] {
		log.Printf("server [%d] listen: %s\n", s.id, s.addr)
	}
	for s.state {
		conn, err := listen.AcceptTCP()
		if err != nil {
			return err
		}
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
	if len(s.connList.GetMap()) > s.maxConnNum {
		_ = write(conn, encodeErrReplyMsgData(ErrServerConnectOverFlow, nil))
		_ = conn.Close()
		return
	}
	am := new(authMsg)
	err := am.unmarshal(conn)
	if err != nil {
		_ = write(conn, encodeErrReplyMsgData(err, nil))
		_ = conn.Close()
		return
	}
	if am.version != Version || am.dstId != s.id || am.srcId == 0 {
		_ = write(conn, encodeErrReplyMsgData(fmt.Errorf("%v -3 ", ErrInvalidConnect), nil))
		_ = conn.Close()
		return
	}
	authData, err := s.authConnect(am)
	if err != nil {
		_ = write(conn, encodeErrReplyMsgData(err, authData))
		_ = conn.Close()
		return
	}
	if err = write(conn, encodeErrReplyMsgData(nil, authData)); err != nil {
		_ = conn.Close()
		log.Println("auth err: ", err)
		return
	}
	sConn := newSrvConn(am.srcId, conn, s)
	s.connList.Set(am.srcId, sConn)
	err = s.gPool.Submit(sConn.Start)
	if err != nil {
		log.Println("ants pool err: ", err)
	}
}

// authConnect authentication connect
func (s *Server) authConnect(auth *authMsg) ([]byte, error) {
	_, ok := s.getConnect(auth.srcId)
	if ok {
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
				hi, ok := s.localHandle.Get(ctx.msg.api)
				if ok {
					hi(newContext(ctx.msg, ctx.conn))
					return
				}
				i, ok := s.registerHandleApiConn.Get(ctx.msg.api)
				if !ok {

					_ = ctx.conn.reply(ctx.msg, msgType_Reply, []byte(ErrNoApi.Error()))
					return
				}
				conn := i.(*srvConn)
				if conn == nil || !conn.Status {
					_ = ctx.conn.reply(ctx.msg, msgType_Reply, []byte(ErrNoApi.Error()))
					return
				}
				ctx.msg.dstId = conn.Id
				if err := conn.writeMsg(ctx.msg); err != nil {
					log.Println("conn.process.localHandle.forward err: ", err)
				}
			default: //转发处理
				conn, ok := s.getConnect(ctx.msg.dstId)
				if !ok {
					_ = ctx.conn.reply(ctx.msg, msgType_Reply, []byte(ErrConnNotExist.Error()))
					return
				}
				if err := conn.writeMsg(ctx.msg); err != nil {
					log.Println("conn.process.forward err: ", err)
				}
			}
		case msgType_Reply, msgType_ReplyErr, msgType_RegistrationResp, msgType_TickResp:
			switch ctx.msg.dstId {
			case s.id, 0:
				obj, ok := ctx.conn.msgChan.Get(ctx.msg.id)
				if !ok {
					log.Println("No recipient drop message", ctx.msg.String())
					return
				}
				mChan, ok := obj.(chan *message)
				if !ok || mChan == nil {
					log.Println("message channel close drop message", ctx.msg.String())
					return
				}
				mChan <- ctx.msg
			default:
				conn, ok := s.getConnect(ctx.msg.dstId)
				if ok {
					if err := conn.writeMsg(ctx.msg); err != nil {
						log.Println("srvConn.process.forward err: ", err)
					}
					return
				}
				fmt.Println("drop reply message: ", ctx.msg.String())
			}
		case msgType_Registration:
			var ok bool
			apis := decodeRegistrationApiReq(ctx.msg.data)
			var badApis = make([]uint32, 0, len(apis))
			for _, api := range apis {
				if _, ok = s.localHandle.Get(api); ok {
					badApis = append(badApis, api)
				} else if i, ok := s.registerHandleApiConn.Get(api); ok {
					conn := i.(*srvConn)
					if conn != nil && conn.Status {
						badApis = append(badApis, api)
					}
				}
			}
			if len(badApis) > 0 {
				_ = ctx.conn.reply(ctx.msg, msgType_RegistrationResp, encodeRegistrationApiResp(ErrRegistrationApiExist, badApis))
				return
			}
			ctx.conn.apis = apis
			for _, api := range apis {
				s.registerHandleApiConn.Set(api, ctx.conn)
				log.Println("RegisterApi: ", api)
			}
			_ = ctx.conn.reply(ctx.msg, msgType_RegistrationResp, encodeRegistrationApiResp(nil, nil))
		case msgType_Tick:
			_ = ctx.conn.reply(ctx.msg, msgType_TickResp, nil)
		}
	})
}

func (s *Server) ConnectEvent(cet connectEventType, arg ...interface{}) {
	switch cet {
	case connectEventType_Close, connectEventType_TimeOutClose:
		conn, ok := arg[0].(*srvConn)
		if ok && conn != nil {
			for _, api := range conn.apis {
				s.registerHandleApiConn.Delete(api)
			}
			conn.close(arg[1].(bool))
		}
		s.connList.Delete(conn.Id)
		log.Printf("close %s: id %d\n", connectEventMap[cet], conn.Id)
	case connectEventType_processClose:
		err, ok := arg[0].(error)
		log.Printf("close connect err: %v %v %T", ok, err, arg[0])
	default:
		log.Println("invalid event: ", cet)
	}
}

func (s *Server) getConnect(id uint64) (*srvConn, bool) {
	i, ok := s.connList.Get(id)
	if !ok {
		return nil, false
	}
	conn := i.(*srvConn)
	if conn == nil || !conn.Status {
		return nil, false
	}
	return conn, true
}

func (s *Server) checkUp() {
	for s.state {
		time.Sleep(s.connKeepAliveTime)
		keys := s.connList.KeyToSlice()
		if len(keys) == 0 {
			continue
		}
		for _, id := range keys {
			conn, ok := s.getConnect(id)
			if ok {
				if time.Now().Unix() > conn.activation+int64(s.connKeepAliveTime.Seconds()) {
					log.Println("超时一个连接关闭", id)
					conn.Close(true)
				}
			}
		}
	}
}

func (s *Server) GetConnect(id uint64) (ISrvConn, bool) {
	return s.getConnect(id)
}

func (s *Server) GetConnList() []ISrvConn {
	key := s.connList.KeyToSlice()
	list := make([]ISrvConn, 0, len(key))
	for _, id := range key {
		conn, ok := s.getConnect(id)
		if ok {
			list = append(list, conn)
		}
	}
	return list
}

func (s *Server) Shutdown() {
	s.state = false
}

func WithSrvId(id uint64) Option {
	return func(srv *Server) error {
		srv.id = id
		return nil
	}
}

func WithSrvConnTimeout(t time.Duration) Option {
	return func(srv *Server) error {
		srv.connKeepAliveTime = t
		return nil
	}
}

func WithSrvGoroutine(min, max int) Option {
	return func(srv *Server) error {
		srv.maxGoroutine = max
		srv.minGoroutine = min
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

// WithSrvMaxConnectNum <= 0 disable The number of connections is not limited
func WithSrvMaxConnectNum(maxNum int) Option {
	return func(srv *Server) error {
		srv.maxConnNum = maxNum
		return nil
	}
}

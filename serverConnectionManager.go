package node

import (
	"errors"
	jeans "github.com/Li-giegie/go-jeans"
	utils "github.com/Li-giegie/go-utils"
	"github.com/panjf2000/ants/v2"
	"log"
	"net"
	"time"
)

// ConnectionEnableFunc  钩子函数 身份验证通过、连接启用后回调
type ConnectionEnableFunc func(conn ISrvConn)

type serverConnectionManagerI interface {
	GetServerConnectionManager() *serverConnectionManager
}

// serverConnectionManager
type serverConnectionManager struct {
	minGoroutine int
	maxGoroutine int
	connTimeOut  time.Duration
	connList     *utils.MapUint64
	ServerI
	AuthenticationFunc
	ConnectionEnableFunc
	LocalHandleFuncList *Handler
	RegisterApiConn     *utils.MapUint32
	*ants.PoolWithFunc
}

// newServerConnectionManager 创建一个默认配置的连接管理器
func newServerConnectionManager(si ServerI) *serverConnectionManager {
	srcConList := new(serverConnectionManager)
	srcConList.connList = utils.NewMapUint64()
	srcConList.LocalHandleFuncList = NewHandler()
	srcConList.RegisterApiConn = utils.NewMapUint32()
	srcConList.maxGoroutine = DEFAULT_MAX_GOROUTINE
	srcConList.minGoroutine = DEFAULT_MIN_GOROUTINE
	srcConList.connTimeOut = DEFAULT_KeepAlive
	srcConList.ConnectionEnableFunc = func(conn ISrvConn) {}
	srcConList.ServerI = si
	return srcConList
}

func (s *serverConnectionManager) GetServerConnectionManager() *serverConnectionManager {
	return s
}

// init 初始化
func (s *serverConnectionManager) init() error {
	poolFunc, err := ants.NewPoolWithFunc(s.minGoroutine, s.handle)
	if err != nil {
		return err
	}
	s.PoolWithFunc = poolFunc
	s.PoolWithFunc.Tune(s.maxGoroutine)
	go s.checkUp()
	return nil
}

func (s *serverConnectionManager) handle(i interface{}) {
	ctx := i.(*srvConnCtx)
	switch ctx.msg.typ {
	case msgType_Send:
		switch ctx.msg.dstId {
		case s.ServerId(), 0: //本地处理
			hi, ok := s.LocalHandleFuncList.Get(ctx.msg.api)
			//本地存在执行，不存在执行转发
			if ok {
				hi(newContext(ctx.msg, ctx.conn))
				return
			}
			conn, ok := s.GetRegisterApiConn(ctx.msg.api)
			if !ok {
				_ = ctx.conn.reply(ctx.msg, msgType_Reply, []byte(ErrNoApi.Error()))
				return
			}
			if err := conn.writeMsg(ctx.msg); err != nil {
				log.Println("conn.process.localHandle.forward err: ", err)
			}
		default: //转发处理
			conn, ok := s.GetConnect(ctx.msg.dstId)
			if !ok || conn == nil || !conn.Status {
				_ = ctx.conn.reply(ctx.msg, msgType_Reply, []byte(ErrConnNotExist.Error()))
				return
			}
			if err := conn.writeMsg(ctx.msg); err != nil {
				log.Println("conn.process.forward err: ", err)
			}
		}
	case msgType_Reply, msgType_RegistrationResp, msgType_TickResp:
		switch ctx.msg.dstId {
		case s.ServerId(), 0:
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
			conn, ok := s.GetConnect(ctx.msg.dstId)
			if !ok || conn == nil || !conn.Status {
				_ = ctx.conn.reply(ctx.msg, msgType_Reply, []byte(ErrConnNotExist.Error()))
				return
			}
			if err := conn.writeMsg(ctx.msg); err != nil {
				log.Println("srvConn.process.forward err: ", err)
			}
		}
	case msgType_Registration:
		var ok bool
		apis := decodeRegistrationApiReq(ctx.msg.data)
		var badApis = make([]uint32, 0, len(apis))
		for _, api := range apis {
			if _, ok = s.LocalHandleFuncList.Get(api); ok {
				badApis = append(badApis, api)
			} else if conn, ok := s.GetRegisterApiConn(api); ok && conn.Status {
				badApis = append(badApis, api)
			}
		}
		if len(badApis) > 0 {
			_ = ctx.conn.reply(ctx.msg, msgType_RegistrationResp, encodeRegistrationApiResp(ErrRegistrationApiExist, badApis))
			return
		}
		ctx.conn.apis = apis
		for _, api := range apis {
			s.RegisterApiConn.Set(api, ctx.conn)
		}
		_ = ctx.conn.reply(ctx.msg, msgType_RegistrationResp, encodeRegistrationApiResp(nil, nil))
	case msgType_Tick:
		_ = ctx.conn.reply(ctx.msg, msgType_TickResp, nil)
	}
}

func (s *serverConnectionManager) GetRegisterApiConn(api uint32) (*srvConn, bool) {
	v, ok := s.RegisterApiConn.Get(api)
	if !ok {
		return nil, false
	}
	conn := v.(*srvConn)
	if conn == nil || !conn.Status {
		return nil, false
	}
	return conn, true
}

func (s *serverConnectionManager) GetConnect(id uint64) (*srvConn, bool) {
	v, ok := s.connList.Get(id)
	if !ok {
		return nil, false
	}
	conn := v.(*srvConn)
	if conn == nil || !conn.Status {
		return nil, false
	}
	return conn, true
}

func (s *serverConnectionManager) ConnectEvent(cet ConnectEventType, arg interface{}) {
	switch cet {
	case ConnectEventType_Close:

	}
}

func (s *serverConnectionManager) addConnect(conn *net.TCPConn) {
	id, err := s.authentication(conn)
	if err != nil {
		_ = conn.Close()
		return
	}
	sConn := newSrvConn(id, conn, s)
	s.connList.Set(id, sConn)
	s.ConnectionEnableFunc(sConn)
	go sConn.Start()
}

// authentication 认证连接是否合法
func (s *serverConnectionManager) authentication(conn *net.TCPConn) (uint64, error) {
	authData, err := jeans.Unpack(conn)
	if err != nil {
		return 0, auth_err_illegality
	}
	id, data := decodeAuthReq(authData)
	if id == 0 || id == s.ServerId() {
		_ = write(conn, encodeAuthResp(nil, auth_err_user_online))
		return id, auth_err_user_online
	}
	iConn, ok := s.connList.Get(id)
	if ok && iConn.(*srvConn).Status {
		_ = write(conn, encodeAuthResp(nil, auth_err_user_online))
		return id, auth_err_user_online
	}

	if s.AuthenticationFunc == nil {
		return id, write(conn, encodeAuthResp(nil, nil))
	}

	ok, b := s.AuthenticationFunc(id, data)
	if !ok {
		err = errors.New(auth_err_head + string(b))
		_ = write(conn, encodeAuthResp(b, err))
		return id, err
	}
	return id, write(conn, encodeAuthResp(b, nil))
}

func (s *serverConnectionManager) write(msg *message) error {
	intfc, ok := s.connList.Get(msg.dstId)
	if !ok {
		return ErrConnNotExist
	}
	conn := intfc.(*srvConn)
	if !conn.Status {
		return ErrDisconnect
	}
	if err := conn.writeMsg(msg); err != nil {
		return ErrDisconnect
	}
	return nil
}

func (s *serverConnectionManager) CloseAllConn() {
	s.connList.RWMutex.Lock()
	defer s.connList.RWMutex.Unlock()
	for _, conn := range s.connList.GetMap() {
		conn.(*srvConn).Close()
	}
}

func (s *serverConnectionManager) CloseConn(id uint64) {
	intfc, ok := s.connList.Get(id)
	if ok {
		intfc.(*srvConn).Close()
	}
}

func (s *serverConnectionManager) checkUp() {
	var invalidConn []*srvConn
	var l int
	for {
		time.Sleep(s.connTimeOut)
		l = len(s.connList.GetMap())
		if l == 0 {
			continue
		}
		invalidConn = make([]*srvConn, 0, l/10+1)
		s.connList.Range(func(k uint64, v interface{}) {
			conn := v.(*srvConn)
			if checkUpTimeOut(time.Duration(conn.activation)*time.Second, s.connTimeOut) {
				invalidConn = append(invalidConn, conn)
			}
		})
		for _, conn := range invalidConn {
			conn.Close()
		}
	}
}

func (s *serverConnectionManager) FindConn(id uint64) (ISrvConn, bool) {
	intfc, ok := s.connList.Get(id)
	if !ok {
		return nil, false
	}
	return intfc.(*srvConn), ok
}

func (s *serverConnectionManager) ConnList() []ISrvConn {
	list := make([]ISrvConn, 0, 10)
	for _, conn := range s.connList.GetMap() {
		list = append(list, conn.(*srvConn))
	}
	return list
}

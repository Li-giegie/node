package node

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	utils "github.com/Li-giegie/go-utils"
	"github.com/panjf2000/ants/v2"
	"log"
	"net"
	"sync"
	"time"
)

type Conn interface {
	Send(api uint32, data []byte) error
	Close(fast ...bool)
	Request(ctx context.Context, api uint32, data []byte) ([]byte, error)
	Id() uint64
}

// ConnectionEnableFunc  钩子函数 身份验证通过、连接启用后回调
type ConnectionEnableFunc func(conn Conn)

type serverConnectionManagerI interface {
	GetServerConnectionManager() *serverConnectionManager
}

type serverI interface {
	getId() uint64
}

// serverConnectionManager
type serverConnectionManager struct {
	minGoroutine    int
	maxGoroutine    int
	connTimeOut     time.Duration
	connList        *utils.MapUint64
	registrationApi *utils.MapUint32
	serverI
	AuthenticationFunc
	ConnectionEnableFunc
	*Handler
	*ants.Pool
}

// newServerConnectionManager 创建一个默认配置的连接管理器
func newServerConnectionManager(si serverI) *serverConnectionManager {
	srcConList := new(serverConnectionManager)
	srcConList.connList = utils.NewMapUint64()
	srcConList.Handler = newHandler()
	srcConList.maxGoroutine = DEFAULT_MAX_GOROUTINE
	srcConList.minGoroutine = DEFAULT_MIN_GOROUTINE
	srcConList.connTimeOut = DEFAULT_KeepAlive
	srcConList.ConnectionEnableFunc = func(conn Conn) {}
	srcConList.registrationApi = utils.NewMapUint32()
	srcConList.serverI = si
	return srcConList
}

func (s *serverConnectionManager) GetServerConnectionManager() *serverConnectionManager {
	return s
}

// init 初始化
func (s *serverConnectionManager) init() error {
	pool, err := ants.NewPool(s.minGoroutine)
	pool.Tune(s.maxGoroutine)
	if err != nil {
		return err
	}
	s.Pool = pool
	for u, _ := range s.handle {
		s.registrationApi.Set(u, s.serverI.getId())
	}
	go s.checkUp()
	return nil
}

func (s *serverConnectionManager) addConnect(conn *net.TCPConn) {
	id, err := s.authentication(conn)
	if err != nil {
		_ = conn.Close()
		return
	}
	srvConn := newServerConnect(id, conn, s)
	s.connList.Set(id, srvConn)
	s.ConnectionEnableFunc(srvConn)
}

// authentication 认证连接是否合法
func (s *serverConnectionManager) authentication(conn *net.TCPConn) (uint64, error) {
	buf, err := jeans.Unpack(conn)
	if err != nil || len(buf) < 8 {
		_ = write(conn, []byte(auth_err_illegality.Error()))
		return 0, auth_err_illegality
	}
	id := binary.LittleEndian.Uint64(buf[:8])
	var data []byte
	if len(buf) > 8 {
		data = buf[8:]
	}
	intfc, ok := s.connList.Get(id)
	if s.serverI.getId() == id || ok && intfc.(*serverConnect).state {
		_ = write(conn, []byte(auth_err_user_online.Error()))
		return id, auth_err_user_online
	}

	if s.AuthenticationFunc == nil {
		err = write(conn, []byte(auth_sucess))
		return id, err
	}

	ok, b := s.AuthenticationFunc(id, data)
	if !ok {
		err = errors.New(auth_err_head + string(b))
		_ = write(conn, []byte(err.Error()))
		return id, err
	}
	err = write(conn, append([]byte(auth_sucess), b...))
	return id, err
}

func (s *serverConnectionManager) write(msg *message) error {
	intfc, ok := s.connList.Get(msg.dstId)
	if !ok {
		return ErrConnNotExist
	}
	conn := intfc.(*serverConnect)
	if !conn.state {
		return ErrDisconnect
	}
	if err := conn.write(msg); err != nil {
		return ErrDisconnect
	}
	return nil
}

func (s *serverConnectionManager) CloseAllConn() {
	s.connList.RWMutex.Lock()
	defer s.connList.RWMutex.Unlock()
	for _, conn := range s.connList.GetMap() {
		conn.(*serverConnect).Close()
	}
}

func (s *serverConnectionManager) CloseConn(id uint64) {
	intfc, ok := s.connList.Get(id)
	if ok {
		intfc.(*serverConnect).Close()
	}
}

func (s *serverConnectionManager) checkUp() {
	var invalidConn []*serverConnect
	var l int
	for {
		time.Sleep(time.Second * 5)
		l = len(s.connList.GetMap())
		if l == 0 {
			continue
		}
		invalidConn = make([]*serverConnect, 0, l/10+1)
		s.connList.Range(func(k uint64, v interface{}) {
			conn := v.(*serverConnect)
			if time.Now().Add(time.Duration(-conn.activate)).UnixNano() > s.connTimeOut.Nanoseconds() {
				invalidConn = append(invalidConn, conn)
			}
		})
		for _, connect := range invalidConn {
			connect.Close()
		}
	}
}

func (s *serverConnectionManager) FindConn(id uint64) (Conn, bool) {
	intfc, ok := s.connList.Get(id)
	if !ok {
		return nil, false
	}
	return intfc.(*serverConnect), ok
}

func (s *serverConnectionManager) ConnList() []Conn {
	list := make([]Conn, 0, len(s.handle))
	for _, conn := range s.connList.GetMap() {
		list = append(list, conn.(*serverConnect))
	}
	return list
}

type serverConnect struct {
	id       uint64
	conn     *net.TCPConn
	activate int64
	state    bool
	response sync.Map
	apiList  []uint32
	serverConnectionManagerI
}

func newServerConnect(id uint64, conn *net.TCPConn, scm serverConnectionManagerI) *serverConnect {
	srvConn := new(serverConnect)
	srvConn.conn = conn
	srvConn.activate = time.Now().UnixNano()
	srvConn.id = id
	srvConn.state = true
	srvConn.serverConnectionManagerI = scm
	go srvConn.process(scm.GetServerConnectionManager().Pool)
	return srvConn
}

func (c *serverConnect) getId() uint64 {
	return c.id
}

func (c *serverConnect) process(p *ants.Pool) {
	defer c.Close()
	for c.state {
		tmp, err := readMessage(c.conn)
		if err != nil {
			c.state = false
			log.Println("readMessage err: ", err)
			break
		}
		msg := tmp
		c.activate = time.Now().UnixNano()
		err = p.Submit(func() {
			switch msg.typ {
			case msgType_Registration: //注册API接受消息
				apiList, err := serverConnectHandleRegistration(c, msg)
				if err == nil {
					c.apiList = apiList
				}
			case msgType_RespSuccess, msgType_RespFail: //服务端发起请求类型
				v, ok := c.response.Load(msg.id)
				if !ok {
					log.Println("Receive timeout message or push message:", msg.String())
					break
				}
				msgChan := v.(chan *message)
				if msgChan != nil {
					msgChan <- msg
				}
			case msgType_Send:
				handler, ok := c.GetServerConnectionManager().handle[msg.api]
				if ok {
					_, _ = handler(msg.srcId, msg.data)
					return
				}
				v, ok := c.GetServerConnectionManager().registrationApi.Get(msg.api)
				if !ok {
					msg.typ = msgType_RespFail
					msg.data = []byte(ErrNoApi.Error())
					_ = c.write(msg)
					break
				}
				msg.srcId = c.id
				msg.typ = msgType_Forward
				msg.dstId = v.(uint64)
				if err = c.GetServerConnectionManager().write(msg); err != nil {
					msg.typ = msgType_RespFail
					msg.data = []byte(err.Error())
					_ = c.write(msg)
				}
			case msgType_Req:
				handler, ok := c.GetServerConnectionManager().handle[msg.api]
				if !ok {
					v, ok := c.GetServerConnectionManager().registrationApi.Get(msg.api)
					if !ok {
						msg.typ = msgType_RespFail
						msg.data = []byte(ErrNoApi.Error())
						_ = c.write(msg)
						break
					}
					msg.srcId = c.id
					msg.typ = msgType_Forward
					msg.dstId = v.(uint64)
					if err = c.GetServerConnectionManager().write(msg); err != nil {
						msg.typ = msgType_RespFail
						msg.data = []byte(err.Error())
						_ = c.write(msg)
					}
					return
				}
				data, err := handler(msg.srcId, msg.data)
				if err != nil {
					msg.typ = msgType_RespFail
					msg.data = []byte(err.Error())
				} else {
					msg.data = data
					msg.typ = msgType_RespSuccess
				}
				_ = c.write(msg)
			case msgType_Forward, msgType_ForwardSuccess, msgType_ForwardFail:
				err = c.GetServerConnectionManager().write(msg)
				if err != nil && msg.typ == msgType_Forward {
					msg.typ = msgType_ForwardFail
					msg.data = []byte(err.Error())
					_ = c.write(msg)
				}
			case msgType_Tick:
				msg.typ = msgType_TickResp
				_ = c.write(msg)
			default:
				fmt.Println("default handle:", msg.String())
				break
			}
		})
		if err != nil {
			log.Println("process err: ", err)
		}
	}
}

func (c *serverConnect) write(m *message) error {
	if !c.state {
		return errors.New("connect state offline")
	}
	if err := writeMsg(c.conn, m); err != nil {
		c.Close()
		return err
	}
	return nil
}

func (c *serverConnect) Request(ctx context.Context, api uint32, data []byte) ([]byte, error) {
	m := newMsgWithReq(api, data)
	replyChan := make(chan *message)
	c.response.Store(m.id, replyChan)
	defer func() {
		c.response.Delete(m.id)
		close(replyChan)
	}()
	if err := c.write(m); err != nil {
		return nil, err
	}
	replyMsg := msgPool.Get().(*message)
	select {
	case replyMsg = <-replyChan:
		if replyMsg.typ == msgType_RespFail {
			return nil, errors.New(string(replyMsg.data))
		}
		return replyMsg.data, nil
	case <-ctx.Done():
		return nil, errors.New("time out")
	}
}

func (c *serverConnect) Id() uint64 {
	return c.id
}

func (c *serverConnect) Send(api uint32, data []byte) error {
	return c.write(newMsgWithSend(api, data))
}

// Close 断开连接，可选参数：如果为true：将立即关闭连接不管发送中的数据是否发送完成
func (c *serverConnect) Close(fast ...bool) {
	if c.conn != nil {
		c.state = false
		if len(fast) > 0 && fast[0] {
			_ = c.conn.SetLinger(0)
		}
		_ = c.conn.Close()
		c.conn = nil
		scm := c.serverConnectionManagerI.GetServerConnectionManager()
		scm.connList.Delete(c.id)
		for _, u := range c.apiList {
			log.Println("close client ", u)
			scm.registrationApi.Delete(u)
		}
	}
}

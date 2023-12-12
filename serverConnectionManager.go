package node

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	utils "github.com/Li-giegie/go-utils"
	"github.com/panjf2000/ants/v2"
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

// serverConnectionManager
type serverConnectionManager struct {
	minGoroutine int
	maxGoroutine int
	connTimeOut  time.Duration
	lock         sync.RWMutex
	connList     map[uint64]*serverConnect
	AuthenticationFunc
	ConnectionEnableFunc
	*Handler
	*ants.PoolWithFunc
}

// newServerConnectionManager 创建一个默认配置的连接管理器
func newServerConnectionManager() *serverConnectionManager {
	srcConList := new(serverConnectionManager)
	srcConList.connList = make(map[uint64]*serverConnect)
	srcConList.Handler = newHandler()
	srcConList.maxGoroutine = DEFAULT_MAX_GOROUTINE
	srcConList.minGoroutine = DEFAULT_MIN_GOROUTINE
	srcConList.connTimeOut = DEFAULT_KeepAlive
	srcConList.ConnectionEnableFunc = func(conn Conn) {}
	return srcConList
}

func (s *serverConnectionManager) GetServerConnectionManager() *serverConnectionManager {
	return s
}

// init 初始化
func (s *serverConnectionManager) init() error {
	pool, err := ants.NewPoolWithFunc(s.minGoroutine, srvConnHandle(s.Handler, s.write))
	if err != nil {
		return err
	}
	pool.Tune(s.maxGoroutine)
	s.PoolWithFunc = pool
	go s.connHealthDetection(s.connTimeOut)
	return nil
}

func (s *serverConnectionManager) addConnect(conn *net.TCPConn) {
	id, err := s.authentication(conn)
	if err != nil {
		_ = conn.Close()
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	srvConn := newServerConnect(id, conn, s.delContList, s.Invoke, s)
	s.connList[id] = srvConn
	s.ConnectionEnableFunc(srvConn)
}

func (s *serverConnectionManager) delContList(id uint64) {
	s.lock.Lock()
	delete(s.connList, id)
	s.lock.Unlock()

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
	s.lock.RLock()
	mgConn, ok := s.connList[id]
	s.lock.RUnlock()

	if ok && mgConn.state {
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
	s.lock.RLock()
	defer s.lock.RUnlock()
	conn, ok := s.connList[msg.dstId]
	if !ok || !conn.state {
		return ErrConnNotExist
	}
	if err := conn.write(msg); err != nil {
		return ErrDisconnect
	}
	return nil
}

func (s *serverConnectionManager) closeAllConn() {
	for _, v := range s.connList {
		v.Close()
	}
}

func (s *serverConnectionManager) connHealthDetection(connTimeOut time.Duration) {
	tick := time.NewTicker(time.Second * 2)
	for range tick.C {
		for s2, connect := range s.connList {
			if connect.activate+int64(connTimeOut.Seconds()) < time.Now().Unix() {
				s.CloseConn(s2)
				fmt.Printf("close connect :[%v]", connect.id)
			}
		}
	}
}

func (s *serverConnectionManager) CloseConn(id uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()
	v, ok := s.connList[id]
	if ok {
		v.Close()
	}
}

func (s *serverConnectionManager) FindConn(id uint64) (Conn, bool) {
	v, ok := s.connList[id]
	return v, ok
}

func (s *serverConnectionManager) ConnList() []Conn {
	list := make([]Conn, 0, len(s.handle))
	for _, conn := range s.connList {
		list = append(list, conn)
	}
	return list
}

type serverConnect struct {
	id          uint64
	conn        *net.TCPConn
	activate    int64
	state       bool
	delContList func(id uint64)
	response    sync.Map
	weight      uint8
	serverConnectionManagerI
	utils.MapUint32I
}

func newServerConnect(id uint64, conn *net.TCPConn, delContList func(id uint64), invoke func(i interface{}) error, scm serverConnectionManagerI) *serverConnect {
	srvConn := new(serverConnect)
	srvConn.conn = conn
	srvConn.activate = time.Now().UnixNano()
	srvConn.id = id
	srvConn.state = true
	srvConn.delContList = delContList
	srvConn.MapUint32I = utils.NewMapUint32()
	srvConn.serverConnectionManagerI = scm
	go srvConn.process(invoke)
	return srvConn
}

func (c *serverConnect) process(invoke func(i interface{}) error) {
	defer func() {
		c.Close()
	}()
	for c.state {
		msg, err := readMessage(c.conn)
		if err != nil {
			break
		}
		c.activate = time.Now().UnixNano()
		switch msg.typ {
		case msgType_Registration:
			var apiList []uint32
			if err = jeans.DecodeSlice(msg.data, &apiList); err != nil {

			}
		default:
			_ = invoke(&nodeContext{
				message:     msg,
				write:       c.write,
				setRespChan: c.response.Load,
			})
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
	if c.state {
		c.state = false
		if len(fast) > 0 && fast[0] {
			_ = c.conn.SetLinger(0)
		}
		_ = c.conn.Close()
		c.delContList(c.id)
	}
}

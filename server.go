package node

import (
	"fmt"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/utils"
	"github.com/panjf2000/ants/v2"
	"log"
	"net"
	"time"
)

// Server 结构
type Server interface {
	//Serve 开启服务
	Serve() error
	//HandleFunc 处理方法
	HandleFunc(api uint16, f common.HandleFunc) bool
	Close() error
	//SetReceiverPoolSize 接收消息管道复用，减轻gc压力
	SetReceiverPoolSize(n int)
	//SetConstructorPoolSize 发送消息结构体复用，构造器容量，消息复用减轻gc压力，n>0 发送完毕的消息会被放入构造器池等待下一次使用
	SetConstructorPoolSize(n int)
	//SetGoroutinePoolSize 最大开启的协程数
	SetGoroutinePoolSize(n int) (err error)
	//SetMaxConnSize 最大连接数
	SetMaxConnSize(n int)
	//SetAuthenticationTimeout 连接认证超时，在时间内没有收到认证内容关闭连接
	SetAuthenticationTimeout(t time.Duration)
	//SetMaxReceiveMsgLength 最大接受消息长度，防范内存溢出 n > 0 启用 <=0 不启用
	SetMaxReceiveMsgLength(n int)
	//Tick 发送一个心跳包，维持连接活跃。(interval：每隔多久检测一次连接是否超时, keepAlive：单位时间后没有收发消息，发送一次心跳包，timeoutClose：单位时间后没有收到心跳包，主动发起关闭连接，showTickPack：可选参数非空index 0为true则把心跳包打印输出到控制台，通常用于测试阶段)
	Tick(interval, keepAlive, timeoutClose time.Duration, showTickPack ...bool) error
	//GetConn 获取服务端已建立的连接
	GetConn(id uint16) (common.SrvConn, bool)
	//GetConns 获取服务端所有建立的连接 可对其进行广播消息
	GetConns() []common.SrvConn
}

type ServerStateType uint8

const (
	ServerStateTypeClose ServerStateType = iota
	ServerStateTypeListen
	ServerStateTypeErr
)

type server struct {
	id                    uint16
	authenticationKey     []byte
	maxConnSize           int
	State                 ServerStateType
	authenticationTimeout time.Duration
	maxReceiveMsgLength   uint32
	net.Listener
	*common.Conns
	*common.Receiver
	*common.Constructor
	*common.ServeMux
	*ants.Pool
}

// NewServer 创建一个Server类型的节点
func NewServer(id uint16, l net.Listener) Server {
	srv := new(server)
	srv.id = id
	srv.Listener = l
	srv.maxConnSize = common.DEFAULT_Max_Conn_Size
	srv.Pool = common.DEFAULT_ServerAntsPool
	srv.Conns = common.DEFAULT_Conns
	srv.Constructor = common.DEFAULT_Constructor
	srv.Receiver = common.DEFAULT_Reveiver
	srv.ServeMux = common.DEFAULT_ServeMux
	srv.State = ServerStateTypeListen
	srv.maxReceiveMsgLength = common.DEFAULT_MaxReceiveMsgLength
	srv.authenticationTimeout = common.DEFAULT_AuthenticationTimeout
	return srv
}

func (s *server) SetReceiverPoolSize(n int) {
	s.Receiver = common.NewMessageReceiver(n)
}

func (s *server) SetAuthenticationTimeout(t time.Duration) {
	s.authenticationTimeout = t
}

func (s *server) SetConstructorPoolSize(n int) {
	s.Constructor = common.NewMessageConstructor(n)
}
func (s *server) SetGoroutinePoolSize(n int) (err error) {
	s.Pool.Release()
	s.Pool, err = ants.NewPool(n)
	return err
}
func (s *server) SetMaxConnSize(n int) {
	s.maxConnSize = n
}

func (s *server) SetMaxReceiveMsgLength(n int) {
	if n <= 0 {
		s.maxReceiveMsgLength = 0
	}
	s.maxReceiveMsgLength = uint32(n)
}

func (s *server) Serve() error {
	i := int64(1)
	d := time.Second
	for {
		if utils.CountSleep(len(s.Conns.M) >= s.maxConnSize, i, d) {
			if i <= 10 {
				i++
			}
			log.Println("Connection pool overflow, exceeding maximum number of connections")
			continue
		}
		i = 1
		conn, err := s.Accept()
		if err != nil {
			return s.checkErr(err)
		}
		err = s.Submit(func() { s.authentication(conn) })
		if err != nil {
			return err
		}
	}
}

func (s *server) authentication(conn net.Conn) {
	err := error(nil)
	code := uint8(0)
	var c common.Conn
	defer func() {
		_, wErr := conn.Write((&common.Authenticator{StateCode: code}).EncodeResp())
		if wErr != nil {
			if err == nil {
				err = wErr
			}
			log.Println("authentication reply ", wErr)
		}
		if err != nil {
			_ = conn.Close()
			if code >= 5 {
				log.Println("authentication err del conn", c.Id())
				s.Conns.Del(c.Id())
			}
			log.Println("authentication err", code, err)
		}
	}()
	auth := new(common.Authenticator)
	err = auth.DecodeReqHeaderWithTimeout(conn, s.authenticationTimeout)
	if err != nil {
		code = 0
		return
	}
	if uint32(len(s.authenticationKey)) != auth.KeyLen {
		code, err = 1, fmt.Errorf("authentication key error")
		return
	}
	if auth.DestId != s.id {
		code, err = 2, fmt.Errorf("authentication destId exist")
		return
	}
	if err = auth.DecodeReqContentWithTimeout(conn, s.authenticationTimeout); err != nil {
		code, err = 3, fmt.Errorf("authentication error")
		return
	}
	if !utils.BytesEqual(auth.Key, s.authenticationKey) {
		code, err = 4, fmt.Errorf("authentication key error")
		return
	}
	c = common.NewConn(s.id, auth.SrcId, conn, s.Constructor, s.Receiver, s, s.maxReceiveMsgLength)
	if !s.Conns.Add(auth.SrcId, c) {
		code, err = 5, fmt.Errorf("authentication error id %d exist", auth.SrcId)
		return
	}
	err = s.Submit(func() {
		err = c.Serve()
		if err != nil {
			log.Println(err)
		}
		s.Conns.Del(c.Id())
	})
	if err != nil {
		code = 6
		return
	}
	code = 200
}

func (s *server) Handle(m *common.Message, c common.Conn) {
	err := s.Submit(func() {
		if m.DestId != 0 && m.DestId != s.id {
			conn, ok := s.Conns.GetConn(m.DestId)
			if !ok {
				m.Reply(common.MsgType_ReplyErrWithConnectNotExist, nil)
				c.WriteMsg(m)
				return
			}
			_, _ = conn.WriteMsg(m)
			return
		}
	})
	if err != nil {
		log.Println("handle err", err)
	}
}

func (s *server) checkErr(err error) error {
	if s.State == ServerStateTypeClose {
		return nil
	}
	s.State = ServerStateTypeErr
	return err
}

func (s *server) Close() error {
	s.State = ServerStateTypeClose
	return s.Listener.Close()
}

type tickPack [common.MESSAGE_HEADER_LEN]byte

func (t *tickPack) Encode() []byte {
	t[0] = common.MsgType_Tick
	return t[:]
}
func (s *server) Tick(interval, keepAlive, timeoutClose time.Duration, showTickPack ...bool) error {
	return s.Submit(func() {
		show := false
		if len(showTickPack) > 0 && showTickPack[0] {
			show = true
		}
		now := int64(0)
		tickBuf := new(tickPack)
		for {
			time.Sleep(interval)
			now = time.Now().UnixMilli()
			conns := s.Conns.GetConns()
			for _, c := range conns {
				if c.State() != common.ConnStateTypeOnConnect {
					continue
				}
				if now >= c.Activate()+timeoutClose.Milliseconds() {
					_ = c.Close()
					if show {
						log.Println("timeout close conn", c.Id())
					}
					return
				} else if now >= c.Activate()+keepAlive.Milliseconds() {
					_, err := c.WriteMsg(tickBuf)
					if err != nil {
						_ = c.Close()
						return
					}
					if show {
						log.Println("send tick --- ", c.Id())
					}
				}
			}

		}
	})
}

func ListenTCP(lid uint16, addr string) (Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewServer(lid, l), nil
}

func ListenAndServeTCP(lid uint16, addr string, h *common.ServeMux) error {
	srv, err := ListenTCP(lid, addr)
	if err != nil {
		return err
	}
	defer srv.Close()
	s := srv.(*server)
	s.ServeMux = h
	return s.Serve()
}

package node

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

type ServerConnectManager struct {
	lock     sync.RWMutex
	connList map[string]*ServerConnect
	ctxChan  chan *Context
	response sync.Map
	count    uint32
}

func newServerConnectManager(ctxChan chan *Context, ConnectionTimeout time.Duration) *ServerConnectManager {
	conn := new(ServerConnectManager)
	conn.connList = make(map[string]*ServerConnect)
	conn.ctxChan = ctxChan
	go conn.connHealthDetection(ConnectionTimeout)
	return conn
}

func (s *ServerConnectManager) addConnect(id string, conn *net.TCPConn) error {
	srvConn := newServerConnect(id, s.closeAndDel, conn)
	s.lock.Lock()
	s.connList[id] = srvConn
	s.lock.Unlock()
	s.count++
	go srvConn.Read(s.ctxChan)
	return nil
}

func (s *ServerConnectManager) write(msg *Message) error {
	conn, ok := s.connList[msg.remoteId]
	if !ok || !conn.state {
		return errors.New("client not exist or offline")
	}
	return conn.write(msg)
}

func (s *ServerConnectManager) closeAllConn() {
	for id, _ := range s.connList {
		s.closeAndDel(id)
	}
}

func (s *ServerConnectManager) connHealthDetection(connTimeOut time.Duration) {
	for {
		for s2, connect := range s.connList {
			if connect.activate+int64(connTimeOut.Seconds()) < time.Now().Unix() {
				s.closeAndDel(s2)
				fmt.Println("检测到关闭")
			}
		}
		time.Sleep(time.Second * 5)
	}
}

func (s *ServerConnectManager) closeAndDel(id string) {
	s.lock.Lock()
	v, ok := s.connList[id]
	if ok {
		if v.state {
			v.state = false
		}
		if v.TCPConn != nil {
			_ = v.TCPConn.Close()
			v.TCPConn = nil
		}
		delete(s.connList, id)
	}
	s.count--
	s.lock.Unlock()
}

type ServerConnect struct {
	id string
	*net.TCPConn
	activate    int64
	state       bool
	closeAndDel func(id string)
}

func newServerConnect(id string, closeAndDel func(id string), conn *net.TCPConn) *ServerConnect {
	srvConn := new(ServerConnect)
	srvConn.TCPConn = conn
	srvConn.activate = time.Now().UnixNano()
	srvConn.id = id
	srvConn.state = true
	srvConn.closeAndDel = closeAndDel
	return srvConn
}

func (c *ServerConnect) Read(ctxChan chan *Context) {
	defer c.Close()
	for c.state {
		msg, err := readMessage(c.TCPConn)
		if err != nil {
			c.state = false
			break
		}
		c.activate = time.Now().UnixNano()
		ctxChan <- NewContext(msg, c.write)
	}
}

func (c *ServerConnect) write(m *Message) error {
	if !c.state {
		return errors.New("connect state offline")
	}
	if err := write(c.TCPConn, m.Marshal()); err != nil {
		c.Close()
		return err
	}
	return nil
}

func (c *ServerConnect) Close() {
	if c.state {
		c.closeAndDel(c.id)
	}
}

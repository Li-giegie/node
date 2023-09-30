package node

import (
	"errors"
	"net"
	"sync"
	"time"
)

type ServerConnectManager struct {
	lock     sync.RWMutex
	connList map[string]*ServerConnect
	ctxChan  chan *Context
	response sync.Map
}

func newServerConnectManager(ctxChan chan *Context) *ServerConnectManager {
	conn := new(ServerConnectManager)
	conn.connList = make(map[string]*ServerConnect)
	conn.ctxChan = ctxChan
	return conn
}

func (s *ServerConnectManager) addConnect(id string, conn *net.TCPConn) error {
	srvConn := newServerConnect(id, conn)
	s.lock.RLock()
	s.connList[id] = srvConn
	s.lock.RUnlock()
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

type ServerConnect struct {
	id string
	*net.TCPConn
	activate int64
	state    bool
}

func newServerConnect(id string, conn *net.TCPConn) *ServerConnect {
	srvConn := new(ServerConnect)
	srvConn.TCPConn = conn
	srvConn.activate = time.Now().UnixNano()
	srvConn.id = id
	srvConn.state = true
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
	c.state = false
	_ = c.TCPConn.Close()
}

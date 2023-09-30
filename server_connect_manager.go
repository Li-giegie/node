package node

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

type ServerConnectManagerI interface {
	getConnect(id string) (*serverConnect, bool)

	setResponse(clientId string, msgId uint32, c chan *MessageBase)
	getResponse(clientId string, msgId uint32) (chan *MessageBase, bool)
	deleteResponse(clientId string, msgId uint32)
}

type ServerConnectManager struct {
	lock       sync.RWMutex
	connList   map[string]*serverConnect
	handleChan chan *Context
	authFunc   AuthenticationFunc
	response   sync.Map
}

func newServerConnectManager(auth AuthenticationFunc, handleChan chan *Context) *ServerConnectManager {
	conn := new(ServerConnectManager)
	conn.connList = make(map[string]*serverConnect)
	conn.authFunc = auth
	conn.handleChan = handleChan
	log.Println("初始化server connect manager ------")
	return conn
}

func (s *ServerConnectManager) addConnect(id string, conn *net.TCPConn) error {
	srvConn := newServerConnect(id, conn)
	s.lock.RLock()
	s.connList[id] = srvConn
	s.lock.RUnlock()
	go srvConn.Read(s.handleChan)
	return nil
}

func (s *ServerConnectManager) write(id string, msg *MessageBase) error {
	conn, ok := s.connList[id]
	if !ok || !conn.state {
		return errors.New("client not exist or offline")
	}
	return conn.write(msg)
}

type serverConnect struct {
	id string
	*net.TCPConn
	activate int64
	state    bool
}

func newServerConnect(id string, conn *net.TCPConn) *serverConnect {
	srvConn := new(serverConnect)
	srvConn.TCPConn = conn
	srvConn.activate = time.Now().UnixNano()
	srvConn.id = id
	srvConn.state = true
	return srvConn
}

func (c *serverConnect) Read(handleChan chan *Context) {
	defer c.Close()
	for c.state {
		msg, err := readMessage(c.TCPConn)
		if err != nil {
			c.state = false
			break
		}
		handleChan <- NewContext(c, msg)
		c.activate = time.Now().UnixNano()
	}
}

func (c *serverConnect) write(m *MessageBase) error {
	if !c.state {
		return nil
	}
	return write(c.TCPConn, m.Marshal())
}

func (c *serverConnect) Close() {
	c.state = false
	_ = c.TCPConn.Close()
}

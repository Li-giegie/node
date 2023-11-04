package node

import (
	"context"
	"errors"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"net"
	"sync"
	"time"
)

type ServerConnectList struct {
	lock     sync.RWMutex
	connList map[uint64]*ServerConnect
	*Handler
	*ants.PoolWithFunc
}

func newServerConnectManager(keepAlive time.Duration, r *Handler, maxGoroutine, minGoroutine int) (*ServerConnectList, error) {
	conn := new(ServerConnectList)
	conn.connList = make(map[uint64]*ServerConnect)
	conn.Handler = r
	pool, err := ants.NewPoolWithFunc(minGoroutine, srvConnHandle(conn.Handler, conn.write))
	if err != nil {
		return nil, err
	}
	pool.Tune(maxGoroutine)
	conn.PoolWithFunc = pool
	go conn.connHealthDetection(keepAlive)
	return conn, nil
}

func (s *ServerConnectList) addConnect(id uint64, conn *net.TCPConn) error {
	srvConn := newServerConnect(id, conn, s.Handler, s.notification, s.Invoke)
	s.lock.Lock()
	defer s.lock.Unlock()
	s.connList[id] = srvConn
	go srvConn.read()
	return nil
}

func (s *ServerConnectList) write(msg *message) error {
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

func (s *ServerConnectList) closeAllConn() {
	for id, _ := range s.connList {
		s.notification(id)
	}
}

func (s *ServerConnectList) notification(id uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()
	v, ok := s.connList[id]
	if ok {
		if v.state {
			v.state = false
		}
		if v.conn != nil {
			v.Close()
			v.conn = nil
		}
		delete(s.connList, id)
	}
}

func (s *ServerConnectList) connHealthDetection(connTimeOut time.Duration) {
	tick := time.NewTicker(time.Second * 2)
	for range tick.C {
		for s2, connect := range s.connList {
			if connect.activate+int64(connTimeOut.Seconds()) < time.Now().Unix() {
				s.notification(s2)
				fmt.Printf("close connect :[%v]", connect.id)
			}
		}
	}
}

type ServerConnect struct {
	id           uint64
	conn         *net.TCPConn
	activate     int64
	state        bool
	notification func(id uint64)
	invoke       func(i interface{}) error
	response     sync.Map
	*Handler
}

func newServerConnect(id uint64, conn *net.TCPConn, r *Handler, notification func(id uint64), invoke func(i interface{}) error) *ServerConnect {
	srvConn := new(ServerConnect)
	srvConn.conn = conn
	srvConn.Handler = r
	srvConn.invoke = invoke
	srvConn.activate = time.Now().UnixNano()
	srvConn.id = id
	srvConn.state = true
	srvConn.notification = notification
	return srvConn
}

func (c *ServerConnect) read() {
	defer c.Close()
	for c.state {
		msg, err := readMessage(c.conn)
		if err != nil {
			c.state = false
			break
		}
		c.activate = time.Now().UnixNano()
		_ = c.invoke(&Context{
			message:     msg,
			write:       c.write,
			setRespChan: c.response.Load,
		})
	}
}

func (c *ServerConnect) write(m *message) error {
	if !c.state {
		return errors.New("connect state offline")
	}
	if err := writeMsg(c.conn, m); err != nil {
		c.Close()
		return err
	}
	return nil
}

func (c *ServerConnect) request(ctx context.Context, api uint32, data []byte) ([]byte, error) {
	m := newMsgWithServReq(api, data)
	replyChan := make(chan *message)
	c.response.Store(m.id, replyChan)
	defer func() {
		c.response.Delete(m.id)
		close(replyChan)
		m.recycle()
	}()
	err := c.write(m)
	if err != nil {
		return nil, err
	}
	select {
	case m = <-replyChan:
		if m._type == MsgType_ServerRespFail {
			err = errors.New(string(m.Data))
		}
		return m.Data, err
	case <-ctx.Done():
		return nil, errors.New("time out")
	}
}

func (c *ServerConnect) send(api uint32, data []byte) error {
	m := newMsgWithSend(api, data)
	err := writeMsg(c.conn, m)
	m.recycle()
	return err
}

func (c *ServerConnect) Close() {
	if c.state {
		_ = c.conn.Close()
		c.notification(c.id)
	}
}

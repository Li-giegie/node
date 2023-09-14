package node

import "net"

type ServerConnectManager struct {
	conn []*serverConnect
}

var srvConnMgmt *ServerConnectManager

func newSrvConnMgmt() *ServerConnectManager {
	srvConnMgmt = new(ServerConnectManager)
	srvConnMgmt.conn = make([]*serverConnect, 0)

	return srvConnMgmt
}

func (m *ServerConnectManager) addConn(conn *net.TCPConn) {
	m.conn = append(m.conn, newServerConnect(conn))
}

func (m *ServerConnectManager) delConn() {

}
func startServerConnectManager() *ServerConnectManager {
	return newSrvConnMgmt()
}

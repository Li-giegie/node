package node

type ServerConnectManager struct {
	conn []*serverConnect
}

var srvConnMgmt *ServerConnectManager

func newSrvConnMgmt() *ServerConnectManager {
	srvConnMgmt = new(ServerConnectManager)
	srvConnMgmt.conn = make([]*serverConnect, 0)

	return srvConnMgmt
}

func startServerConnectManager() *ServerConnectManager {
	return newSrvConnMgmt()
}

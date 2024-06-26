package node

import (
	"github.com/Li-giegie/node/common"
	"net"
)

// Dial network、address，同net.Dail相同，localId 本地节点Id，node 生命周期管理实现接口
func Dial(network, address string, localId uint16, node Node) (common.Conn, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return Serve(conn, localId, node, nil)
}

// Serve 非阻塞，net.Conn已建立的连接，localId 本地节点Id，node 生命周期管理实现接口、conns当前节点需要继承另一个服务端节点做边界网关节点时传入服务端节点，如不需要传入nil
/*
asd
*/
func Serve(conn net.Conn, localId uint16, node Node, conns common.Connections) (common.Conn, error) {
	nodeConn, err := common.NewConn(localId, conn, common.NewMsgPool(1024), common.NewMsgReceiver(1024), conns, node)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	go func() {
		nodeConn.Serve(node)
		_ = nodeConn.Close()
	}()
	return nodeConn, nil
}

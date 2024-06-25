package node

import (
	"github.com/Li-giegie/node/common"
	"net"
)

type Client struct {
	LocalId        uint16
	remoteId       uint16
	MaxMsgLen      uint32
	MsgPoolSize    int
	MsgReceiveSize int
	p              *common.MsgPool
	r              *common.MsgReceiver
}

func Dial(network, address string, localId uint16, node Node) (common.Conn, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	nodeConn, err := common.NewConn(localId, conn, common.NewMsgPool(1024), common.NewMsgReceiver(1024), nil, node)
	if err != nil {
		return nil, err
	}
	go func() {
		nodeConn.Serve(node)
		_ = nodeConn.Close()
	}()
	return nodeConn, nil
}

func Serve(conn net.Conn, localId uint16, node Node) (common.Conn, error) {
	nodeConn, err := common.NewConn(localId, conn, common.NewMsgPool(1024), common.NewMsgReceiver(1024), nil, node)
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

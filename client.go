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
	remoteId, err := node.Connection(conn)
	if err != nil {
		return nil, err
	}
	rev := common.NewMsgReceiver(1024)
	pool := common.NewMsgPool(1024)
	nodeConn := common.NewConn(localId, remoteId, conn, pool, rev, nil, 0x00FFFFFF)
	go func() {
		nodeConn.Serve(node)
		_ = nodeConn.Close()
	}()
	return nodeConn, nil
}

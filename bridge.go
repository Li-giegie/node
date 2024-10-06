package node

import (
	"errors"
	"net"
)

type BridgeNode interface {
	Conn() net.Conn
	RemoteId() uint32
	Disconnection()
}

type bridge struct {
	rid    uint32
	lid    uint32
	disFun func()
	conn   net.Conn
}

func CreateBridgeNode(conn net.Conn, id *Identity, disconnectionFunc func()) (BridgeNode, error) {
	err := defaultBasicReq.Send(conn, id.Id, id.AccessKey)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	rid, permit, msg, err := defaultBasicResp.Receive(conn, id.AccessTimeout)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if !permit {
		_ = conn.Close()
		return nil, errors.New(msg)
	}
	b := new(bridge)
	b.conn = conn
	b.rid = rid
	b.lid = id.Id
	b.disFun = disconnectionFunc
	return b, nil
}

func (b *bridge) Conn() net.Conn {
	return b.conn
}

func (b *bridge) RemoteId() uint32 {
	return b.rid
}

func (b *bridge) Disconnection() {
	if b.disFun != nil {
		b.disFun()
	}
}

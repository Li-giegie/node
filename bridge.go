package node

import (
	"context"
	"net"
)

type BridgeNode interface {
	Conn() net.Conn
	RemoteId() uint16
	Disconnection()
}

type bridge struct {
	rid    uint16
	lid    uint16
	disFun func()
	conn   net.Conn
}

func CreateBridgeNode(ctx context.Context, conn net.Conn, lid, rid uint16, disconnectionFunc func()) (BridgeNode, error) {
	connInit := new(ConnInitializer)
	connInit.LocalId = lid
	connInit.RemoteId = rid
	err := error(nil)
	if err = connInit.Send(conn); err != nil {
		return nil, err
	}
	if err = connInit.ReceptionWithCtx(ctx, conn); err != nil {
		return nil, err
	}
	if err = connInit.Error(); err != nil {
		return nil, err
	}
	b := new(bridge)
	b.conn = conn
	b.rid = rid
	b.lid = lid
	b.disFun = disconnectionFunc
	return b, nil
}

func (b *bridge) Conn() net.Conn {
	return b.conn
}

func (b *bridge) RemoteId() uint16 {
	return b.rid
}

func (b *bridge) Disconnection() {
	if b.disFun != nil {
		b.disFun()
	}
}

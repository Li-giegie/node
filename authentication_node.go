package node

import (
	"net"
)

type ExternalDomainNode interface {
	Conn() net.Conn
	RemoteId() uint16
}

type InitFunc func(conn net.Conn) (remoteId uint16, err error)

type DomainNode struct {
	conn     net.Conn
	remoteId uint16
}

func (u *DomainNode) Conn() net.Conn {
	return u.conn
}

func (u *DomainNode) RemoteId() uint16 {
	return u.remoteId
}

func DialExternalDomainNode(network string, addr string, connection InitFunc) (ExternalDomainNode, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	remoteId, err := connection(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &DomainNode{
		conn:     conn,
		remoteId: remoteId,
	}, nil
}

func NewExternalDomainNode(conn net.Conn, connection InitFunc) (ExternalDomainNode, error) {
	remoteId, err := connection(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &DomainNode{
		conn:     conn,
		remoteId: remoteId,
	}, nil
}

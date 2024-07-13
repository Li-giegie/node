package node

import (
	"net"
)

type AuthenticationNode interface {
	Conn() net.Conn
	RemoteId() uint16
}

type InitFunc func(conn net.Conn) (remoteId uint16, err error)

type authenticationNode struct {
	conn       net.Conn
	remoteId   uint16
	statusSync bool
}

func (u *authenticationNode) Conn() net.Conn {
	return u.conn
}

func (u *authenticationNode) RemoteId() uint16 {
	return u.remoteId
}

func NewAuthenticationNode(network string, addr string, connection InitFunc) (AuthenticationNode, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	remoteId, err := connection(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &authenticationNode{
		conn:       conn,
		remoteId:   remoteId,
		statusSync: false,
	}, nil
}

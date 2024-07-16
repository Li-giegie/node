package authentication

import (
	"net"
	"time"
)

type ServerAuthProtocol struct {
	id      uint16
	key     string
	timeout time.Duration
	*AuthProtocol
}

func NewServerAuthProtocol(id uint16, key string, timeout time.Duration) *ServerAuthProtocol {
	return &ServerAuthProtocol{
		id:           id,
		key:          key,
		timeout:      timeout,
		AuthProtocol: &AuthProtocol{},
	}
}

func (s *ServerAuthProtocol) Init(conn net.Conn) (remoteId uint16, err error) {
	return s.InitServer(conn, s.id, s.key, s.timeout)
}

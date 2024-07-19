package authentication

import (
	"net"
	"time"
)

type ClientAuthProtocol struct {
	id      uint16
	key     string
	timeout time.Duration
	*AuthProtocol
}

func NewClientAuthProtocol(id uint16, key string, timeout time.Duration) *ClientAuthProtocol {
	return &ClientAuthProtocol{
		id:           id,
		key:          key,
		timeout:      timeout,
		AuthProtocol: &AuthProtocol{},
	}
}

func (s *ClientAuthProtocol) Init(conn net.Conn) (remoteId uint16, err error) {
	return s.InitClient(conn, s.id, s.key, s.timeout)
}

package responsewriter

import "github.com/Li-giegie/node/pkg/conn"

type ResponseWriter interface {
	Response(stateCode int16, data []byte) error
	GetConn() conn.Conn
}

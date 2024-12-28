package connmanager

import "github.com/Li-giegie/node/pkg/conn"

type ConnManager interface {
	AddConn(id uint32, conn conn.Conn) bool
	RemoveConn(id uint32)
	GetConn(id uint32) (conn.Conn, bool)
	GetAllConn() []conn.Conn
	RangeConn(f func(conn conn.Conn) bool)
	LenConn() (n int)
}

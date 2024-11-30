package iface

type ConnManager interface {
	AddConn(id uint32, conn Conn) bool
	RemoveConn(id uint32)
	GetConn(id uint32) (Conn, bool)
	GetAllConn() []Conn
	Len() (n int)
}

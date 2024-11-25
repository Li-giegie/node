package iface

type ConnManager interface {
	Add(id uint32, conn Conn) bool
	Remove(id uint32)
	Get(id uint32) (Conn, bool)
	GetAll() []Conn
	Len() (n int)
}

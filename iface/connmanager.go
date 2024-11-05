package iface

type ConnManager interface {
	Add(id uint32, conn Conn) bool
	Remove(id uint32)
	Get(id uint32) (Conn, bool)
	GetAll() []Conn
	GetAllId() []uint32
	Len() (n int)
}

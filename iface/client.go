package iface

type Client interface {
	Start() error
	Close() error
	AddOnConnection(callback func(conn Conn))
	AddOnMessage(callback func(conn Context))
	AddOnCustomMessage(callback func(conn Context))
	AddOnNoIdMessage(callback func(conn Context))
	AddOnClosed(callback func(conn Conn, err error))
	Conn
}

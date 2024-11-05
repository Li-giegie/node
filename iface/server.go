package iface

import (
	"context"
	"net"
	"time"
)

type Server interface {
	Serve() error
	Bridge(conn net.Conn, remoteAuthKey []byte, timeout time.Duration) (rid uint32, err error)
	Request(ctx context.Context, dst uint32, data []byte) ([]byte, error)
	WriteTo(dst uint32, data []byte) (int, error)
	GetConn(id uint32) (Conn, bool)
	GetAllConn() []Conn
	GetAllId() []uint32
	Id() uint32
	Close() error
	Router
	AddOnClosed(callback func(conn Conn, err error))
	AddOnCustomMessage(callback func(conn Context))
	AddOnMessage(callback func(conn Context))
	AddOnConnection(callback func(conn Conn))
}

type Handler interface {
	AddOnClosed(callback func(conn Conn, err error))
	AddOnCustomMessage(callback func(conn Context))
	AddOnMessage(callback func(conn Context))
	AddOnConnection(callback func(conn Conn))
}

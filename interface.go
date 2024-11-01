package node

import (
	"context"
	"github.com/Li-giegie/node/net"
)

type Conn interface {
	Request(ctx context.Context, data []byte) ([]byte, error)
	Forward(ctx context.Context, destId uint32, data []byte) ([]byte, error)
	Write(data []byte) (n int, err error)
	WriteTo(dst uint32, data []byte) (n int, err error)
	WriteMsg(m *net.Message) (n int, err error)
	Close() error
	LocalId() uint32
	RemoteId() uint32
	Activate() int64
}

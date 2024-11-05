package iface

import (
	"context"
	"github.com/Li-giegie/node/message"
)

type Conn interface {
	Request(ctx context.Context, data []byte) ([]byte, error)
	Forward(ctx context.Context, destId uint32, data []byte) ([]byte, error)
	Write(data []byte) (n int, err error)
	WriteTo(dst uint32, data []byte) (n int, err error)
	WriteMsg(m *message.Message) (n int, err error)
	Close() error
	LocalId() uint32
	RemoteId() uint32
	Activate() int64
	NodeType() uint8
	IsClosed() bool
}

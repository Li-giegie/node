package node

import (
	"github.com/Li-giegie/node/pkg/client"
	"github.com/Li-giegie/node/pkg/server"
)

func NewServerOption(localId uint32, opts ...server.Option) server.Server {
	return server.NewServerOption(localId, opts...)
}

func NewServer(localId uint32, c *server.Config) server.Server {
	return server.NewServer(localId, c)
}

func NewClientOption(localId, remoteId uint32, opts ...client.Option) client.Client {
	return client.NewClientOption(localId, remoteId, opts...)
}

func NewClient(localId, remoteId uint32, c *client.Config) client.Client {
	return client.NewClient(localId, remoteId, c)
}

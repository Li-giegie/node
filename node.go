package node

import (
	"github.com/Li-giegie/node/pkg/client"
	"github.com/Li-giegie/node/pkg/server"
)

func NewServerOption(id uint32, opts ...server.Option) server.Server {
	return server.NewServerOption(id, opts...)
}

func NewServer(c *server.Config) server.Server {
	return server.NewServer(c)
}

// NewClientOption 创建客户端，lid本地节点Id，rid远程节点Id
func NewClientOption(localId, remoteId uint32, opts ...client.Option) client.Client {
	return client.NewClientOption(localId, remoteId, opts...)
}

func NewClient(c *client.Config) client.Client {
	return client.NewClient(c)
}

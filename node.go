package node

import (
	"github.com/Li-giegie/node/pkg/client"
	"github.com/Li-giegie/node/pkg/client/impl_client"
	"github.com/Li-giegie/node/pkg/common"
	"github.com/Li-giegie/node/pkg/server"
	"github.com/Li-giegie/node/pkg/server/impl_server"
)

func NewServer(identity *common.Identity, conf ...*common.Config) server.Server {
	return impl_server.NewServer(identity, conf...)
}

func NewClient(localId uint32, remote *common.Identity, conf ...*common.Config) client.Client {
	return impl_client.NewClient(localId, remote, conf...)
}

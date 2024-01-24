package node_register

import (
	utils "github.com/Li-giegie/go-utils"
	"github.com/Li-giegie/node"
)

type IRegister interface {
	ListenAndServe(debug ...bool) error
	Shutdown()
}

func NewRegister(addr string, option ...node.Option) IRegister {
	srvNode := node.NewServer(addr, option...)
	srvNode.HandleFunc(0, func(ctx *node.Context) {
		m := new(message)
		err := m.unmarshal(ctx.Data())
		if err != nil {
			ctx.ReplyErr(err, nil)
		}

		ctx.Reply(nil)
	})
	m32 := utils.NewMapUint64()
	return &RegisterService{IServer: srvNode, MapUint64: m32}
}

type RegisterService struct {
	*utils.MapUint64
	node.IServer
}

func (r *RegisterService) ListenAndServe(debug ...bool) error {
	return r.IServer.ListenAndServer(debug...)
}

func (r *RegisterService) Shutdown() {
	r.IServer.Shutdown()
}

type ServerNode interface {
	Register(weight uint16, id uint64, apiList []uint32)
}

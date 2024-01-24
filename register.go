package node

import (
	utils "github.com/Li-giegie/go-utils"
)

type iRegisterHandle interface {
	QueryRegisterConn(api uint32) (*srvConn, bool)
	DeleteRegisterConn(api uint32)
	AppendRegisterConn(conn *srvConn, apis []uint32)
}

func newRegisterHandle() iRegisterHandle {
	rh := new(registerHandle)
	rh.MapUint32 = utils.NewMapUint32()
	return rh
}

type registerHandle struct {
	*utils.MapUint32
}

func (r *registerHandle) QueryRegisterConn(api uint32) (*srvConn, bool) {
	arg, ok := r.Get(api)
	if !ok {
		return nil, false
	}
	return arg.(*srvConn), true
}

func (r *registerHandle) DeleteRegisterConn(api uint32) {
	r.Delete(api)
}

func (r *registerHandle) AppendRegisterConn(conn *srvConn, apis []uint32) {
	for i, _ := range apis {
		r.Set(apis[i], conn)
	}
}

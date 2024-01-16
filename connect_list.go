package node

import utils "github.com/Li-giegie/go-utils"

type iConnectList interface {
	Add(conn *srvConn)
	Delete(id uint64)
	Query(id uint64) (*srvConn, bool)
	Len() int
	Keys() []uint64
}

func newConnectList() iConnectList {
	cl := new(connectList)
	cl.MapUint64 = utils.NewMapUint64()
	return cl
}

type connectList struct {
	*utils.MapUint64
}

func (c *connectList) Len() int {
	return len(c.GetMap())
}

func (c *connectList) Add(conn *srvConn) {
	c.Set(conn.Id, conn)
}

func (c *connectList) Delete(id uint64) {
	c.MapUint64.Delete(id)
}

func (c *connectList) Query(id uint64) (*srvConn, bool) {
	v, ok := c.Get(id)
	if !ok {
		return nil, false
	}
	return v.(*srvConn), true
}

func (c *connectList) Keys() []uint64 {
	return c.KeyToSlice()
}

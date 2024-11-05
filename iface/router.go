package iface

type Router interface {
	AddRoute(dst, via uint32, hop uint8) (id uint64)
	RemoveRoute(dst, via uint32) bool
	RemoveRouteWithDst(dst uint32)
	RemoveRouteWithVia(via uint32) (affected int)
	RemoveRouteWithId(id uint64) bool
	GetRoute(dst uint32) (via uint32, exist bool)
	GetRouteWithVia(via uint32) (dst []uint32)
	RangeRoute(f func(id uint64, dst uint32, via uint32, hop uint8))
}

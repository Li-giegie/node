package node

import "log"

type SetRouterI interface {
	Api() uint32
	Handler() HandlerFunc
}

type RouteManager struct {
	api         map[uint32]HandlerFunc
	NoApi       HandlerFunc
	TickApi     HandlerFunc
	AbnormalApi HandlerFunc
}

func newRouter() *RouteManager {
	r := new(RouteManager)
	r.api = make(map[uint32]HandlerFunc)
	r.NoApi = defaultNoRouteHandle()
	r.TickApi = defaultTickHandle()
	r.AbnormalApi = defaultAbnormalApiHandle()
	return r
}

func (r *RouteManager) AddRoute(api uint32, handle HandlerFunc) *RouteManager {
	if _, ok := r.api[api]; ok {
		log.Printf("[warning] router api repeat [%v]\n", api)
		return r
	}
	r.api[api] = handle
	return r
}

func (r *RouteManager) AddRouterI(ri ...SetRouterI) *RouteManager {
	for _, _r := range ri {
		r.AddRoute(_r.Api(), _r.Handler())
	}
	return r
}

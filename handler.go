package node

import "log"

type HandleFunc func(id uint64, data []byte) (out []byte, err error)

type HandlerI interface {
	Id() uint32
	HandleFunc() HandleFunc
}

type Handler struct {
	handle map[uint32]HandleFunc
}

func newRouter() *Handler {
	r := new(Handler)
	r.handle = make(map[uint32]HandleFunc)
	return r
}

func (r *Handler) HandleFunc(api uint32, handle HandleFunc) *Handler {
	if _, ok := r.handle[api]; ok {
		log.Printf("[warning] router route repeat [%v]\n", api)
		return r
	}
	r.handle[api] = handle
	return r
}

func (r *Handler) HandlerI(ri ...HandlerI) *Handler {
	for _, _r := range ri {
		if _, ok := r.handle[_r.Id()]; ok {
			log.Printf("duplicate routing [%v]\n", _r.Id())
		}

		r.HandleFunc(_r.Id(), _r.HandleFunc())
	}
	return r
}

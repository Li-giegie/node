package node

import "github.com/panjf2000/ants/v2"

type taskPool struct {
	numWorker int
	*RouteManager
	*ants.PoolWithFunc
}

func newTaskPool(numWorker int, r *RouteManager) {
	t := new(taskPool)
	t.RouteManager = r
	p, _ := ants.NewPoolWithFunc(1, func(i interface{}) {

	})
	t.PoolWithFunc = p
}

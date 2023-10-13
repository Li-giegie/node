package node

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type GoroutineState uint8

const (
	GoroutineStateStop GoroutineState = iota
	GoroutineStateRunning
	//GoroutineState_abnormal
)

type Goroutine struct {
	Id      uint32
	State   GoroutineState
	ctxChan <-chan *Context
	Running bool
	GoroutineManagerI
}

func newGoroutine(id uint32, ctxChan <-chan *Context, gmi GoroutineManagerI) *Goroutine {
	gr := new(Goroutine)
	gr.Id = id
	gr.State = GoroutineStateRunning
	gr.Running = true
	gr.ctxChan = ctxChan
	gr.GoroutineManagerI = gmi
	return gr
}

var l sync.Mutex

func (g *Goroutine) Start() {
	log.Printf("worker process start --- id[%v]\n", g.Id)
	route := g.getRoutes()
	for g.Running {
		ctx := <-g.ctxChan
		if ctx._type == MsgType_Req {
			handler, ok := route.api[ctx.Message.API]
			if !ok {
				ctx._type = MsgType_ReqFail
				ctx.Data = []byte("err: no api")
				_ = ctx.write(ctx.Message)
				continue
			}
			handler(ctx)
		} else if ctx.isForward() {
			err := g.forward(ctx.Message)
			if err != nil && ctx._type == MsgType_ReqForward {
				ctx._type = MsgType_ReqForwardFail
				ctx.Data = []byte("forward err:" + err.Error())
				_ = ctx.write(ctx.Message)
			}
		} else if ctx._type == MsgType_Tick {
			ctx._type = MsgType_TickResp
			_ = ctx.write(ctx.Message)
		} else {
			fmt.Println("default handle:", ctx.Message.String())
		}
	}
	log.Printf("worker process exit --- id[%v]\n", g.Id)
}
func (g *Goroutine) Stop() {
	g.Running = false
}

type GoroutineManagerI interface {
	getRoutes() *RouteManager
	forward(m *Message) error
}

type GoroutineManager struct {
	num        int
	routes     *RouteManager
	count      uint32
	ctxChan    chan *Context
	Goroutines []*Goroutine
	*ServerConnectManager
}

// 创建一个协程管理器
func newGoroutineManager(num int, routes *RouteManager, ctxChan chan *Context, scm *ServerConnectManager) *GoroutineManager {
	gm := new(GoroutineManager)
	gm.Goroutines = make([]*Goroutine, 0, num)
	gm.count = uint32(num)
	gm.ctxChan = ctxChan
	gm.routes = routes
	gm.num = num
	gm.ServerConnectManager = scm
	return gm
}

func (gr *GoroutineManager) getRoutes() *RouteManager {
	return gr.routes
}

func (gr *GoroutineManager) forward(m *Message) error {
	return gr.ServerConnectManager.write(m)
}

func (gr *GoroutineManager) start() {
	for i := 0; i < gr.num; i++ {
		g := newGoroutine(uint32(i+1), gr.ctxChan, gr)
		gr.Goroutines = append(gr.Goroutines, g)
		go g.Start()

	}
	gr.GoroutineDebug()
}

// AddGoroutine 添加一个协程
func (gr *GoroutineManager) AddGoroutine() {
	for i, g2 := range gr.Goroutines {
		if g2.Running == false {
			gr.Goroutines[i].Running = true
			gr.Goroutines[i].State = GoroutineStateRunning
			go gr.Goroutines[i].Start()
			log.Println("GoroutineManager add [restart] Goroutine :", g2.Id)
			return
		}
	}

	id := atomic.AddUint32(&gr.count, 1)
	g := newGoroutine(id, gr.ctxChan, gr)
	gr.Goroutines = append(gr.Goroutines, g)
	log.Println("GoroutineManager add Goroutine :", id)
}

// SubGoroutine 减少一个协程
func (gr *GoroutineManager) SubGoroutine() bool {
	var ok bool
	for _, g2 := range gr.Goroutines {
		if g2.Running {
			g2.Running = false
			g2.State = GoroutineStateStop
			ok = true
			log.Println("GoroutineManager exit Goroutine :", g2.Id)
			break
		}
	}
	return ok
}

func (gr *GoroutineManager) GoroutineAllClose() {
	for _, g2 := range gr.Goroutines {
		g2.Running = false
		g2.State = GoroutineStateStop
	}
}

func (gr *GoroutineManager) GoroutineDebug() {
	go func() {
		var old = -1
		for {
			time.Sleep(time.Second)
			var tmp = make([]uint32, 0, len(gr.Goroutines))
			for _, connect := range gr.Goroutines {
				if connect.Running {
					tmp = append(tmp, connect.Id)
				}
			}
			if len(tmp) == old {
				continue
			}
			old = len(tmp)
			fmt.Println("存活协程：", len(tmp), tmp)
		}
	}()
}

package goroutine_manager

import "log"

type GoroutineState uint8

const (
	GoroutineState_End GoroutineState = iota
	GoroutineState_Start
	GoroutineState_Stop
)

type GoroutineManagerHandleFunc func(arg interface{})

type GoroutineManager struct {
	inputChan      chan interface{}
	handleFunc     GoroutineManagerHandleFunc
	goroutineIndex int
	state          []GoroutineState
}

func NewGoroutineManager(inputChan chan interface{}, handleFunc GoroutineManagerHandleFunc) *GoroutineManager {
	g := new(GoroutineManager)
	g.inputChan = inputChan
	g.handleFunc = handleFunc
	g.state = make([]GoroutineState, 0, cap(inputChan))
	return g
}

func (g *GoroutineManager) Run() {
	log.Println("start Goroutine ")
	for i := 0; i < cap(g.inputChan); i++ {
		go g.AddGoroutine()
	}
}

func (g *GoroutineManager) AddGoroutine() {
	id := g.goroutineIndex
	g.goroutineIndex++
	g.state = append(g.state, GoroutineState_Start)
	log.Printf("goroutine id [%d] start \n", id)
	defer func() {
		if err := recover(); any(err) != nil {
			log.Printf("goroutine id [%d] painc [restarted] error: %v  \n", id, err)
		}
	}()
	for {
		switch g.state[id] {
		case GoroutineState_Start:
			val := <-g.inputChan
			g.handleFunc(val)
		case GoroutineState_Stop:
			continue
		case GoroutineState_End:
			return
		}
	}

}

func (g *GoroutineManager) Stop(id int) {

}

func (g *GoroutineManager) End(id int) {

}

//
//type goroutine struct {
//	Id        uint32
//	State     GoroutineState
//	InputChan <-chan *Context
//	Running   bool
//	gmi       GoroutineManagerI
//}
//
//func newGoroutine(id uint32, inputChan <-chan *Context, gmi GoroutineManagerI, isRun ...bool) *goroutine {
//	gr := new(goroutine)
//	gr.Id = id
//	gr.State = GoroutineState_Running
//	gr.Running = true
//	gr.InputChan = inputChan
//	gr.gmi = gmi
//	if len(isRun) > 0 && isRun[0] == true {
//		go gr.Start()
//	}
//
//	return gr
//}
//
//func (g *goroutine) Start() {
//	log.Printf("worker process start --- id[%v]\n", g.Id)
//	for g.Running {
//		ctx := <-g.InputChan
//		switch ctx.GetType() {
//		case MessageBaseType_Single, MessageBaseType_Request:
//			handler, ok := g.gmi.getHandler(ctx.GetAPI())
//			if !ok {
//				g.gmi.getNoRouteHandle()(ctx)
//				continue
//			}
//			handler(ctx)
//		case MessageBaseType_Tick:
//			g.gmi.getTickHandle()(ctx)
//		case MessageBaseType_RequestForward, MessageBaseType_SingleForward, MessageBaseType_ResponseForward:
//			msg := ctx.get()
//			forwardMsg := NewMessageForwardWithUnmarshal(msg.Data)
//			err := g.gmi.forward(forwardMsg.DestId, msg)
//			if err != nil {
//				forwardMsg.Data = []byte("forward err:" + err.Error())
//				msg.Data = forwardMsg.Marshal()
//				if err = ctx.write(msg); err != nil {
//					ctx.Close()
//				}
//			}
//		default:
//			g.gmi.getAbnormalApi()(ctx)
//		}
//	}
//	log.Printf("worker process exit --- id[%v]\n", g.Id)
//}
//func (g *goroutine) Stop() {
//	g.Running = false
//}
//
//type GoroutineManagerI interface {
//	getTickHandle() HandlerFunc
//	getNoRouteHandle() HandlerFunc
//	getHandler(id uint32) (HandlerFunc, bool)
//	getAbnormalApi() HandlerFunc
//	forward(id string, m *MessageBase) error
//}
//
//type goroutineManager struct {
//	num        int
//	routes     *RouteManager
//	count      uint32
//	inputChan  chan *Context
//	goroutines []*goroutine
//	*ServerConnectManager
//}
//
//// 创建一个协程管理器
//func newGoroutineManager(num int, routes *RouteManager, inputChan chan *Context, scm *ServerConnectManager) *goroutineManager {
//	gm := new(goroutineManager)
//	gm.goroutines = make([]*goroutine, 0, num)
//	gm.count = uint32(num)
//	gm.inputChan = inputChan
//	gm.routes = routes
//	gm.num = num
//	gm.ServerConnectManager = scm
//	return gm
//}
//
//func (g *goroutineManager) getTickHandle() HandlerFunc {
//	return g.routes.TickApi
//}
//
//func (g *goroutineManager) getNoRouteHandle() HandlerFunc {
//	return g.routes.NoApi
//}
//
//func (g *goroutineManager) getAbnormalApi() HandlerFunc {
//	return g.routes.AbnormalApi
//}
//
//func (g *goroutineManager) getHandler(id uint32) (HandlerFunc, bool) {
//	h, ok := g.routes.api[id]
//	return h, ok
//}
//
//func (g *goroutineManager) forward(id string, m *MessageBase) error {
//	return g.ServerConnectManager.write(id, m)
//}
//
//func (g *goroutineManager) start() {
//	for i := 0; i < g.num; i++ {
//		gr := newGoroutine(uint32(i+1), g.inputChan, g, true)
//		g.goroutines = append(g.goroutines, gr)
//	}
//	g.goroutineDebug()
//}
//
//// 添加一个协程
//func (g *goroutineManager) AddGoroutine() {
//	for i, g2 := range g.goroutines {
//		if g2.Running == false {
//			g.goroutines[i].Running = true
//			g.goroutines[i].State = GoroutineState_Running
//			go g.goroutines[i].Start()
//			log.Println("GoroutineManager add [restart] goroutine :", g2.Id)
//			return
//		}
//	}
//
//	id := atomic.AddUint32(&g.count, 1)
//	gr := newGoroutine(id, g.inputChan, g, true)
//	g.goroutines = append(g.goroutines, gr)
//	log.Println("GoroutineManager add goroutine :", id)
//}
//
//// 减少一个协程
//func (g *goroutineManager) SubGoroutine() bool {
//	var ok bool
//	for _, g2 := range g.goroutines {
//		if g2.Running {
//			g2.Running = false
//			g2.State = GoroutineState_Stop
//			ok = true
//			log.Println("GoroutineManager exit goroutine :", g2.Id)
//			break
//		}
//	}
//	return ok
//}
//
//func (g *goroutineManager) GoroutineAllClose() {
//	for _, g2 := range g.goroutines {
//		g2.Running = false
//		g2.State = GoroutineState_Stop
//	}
//}
//
//func (g *goroutineManager) goroutineDebug() {
//	go func() {
//		var old int = -1
//		for {
//			time.Sleep(time.Second)
//			var tmp = make([]uint32, 0, len(g.goroutines))
//			for _, connect := range g.goroutines {
//				if connect.Running {
//					tmp = append(tmp, connect.Id)
//				}
//			}
//			if len(tmp) == old {
//				continue
//			}
//			old = len(tmp)
//			fmt.Println("存活协程：", len(tmp), tmp)
//		}
//	}()
//}

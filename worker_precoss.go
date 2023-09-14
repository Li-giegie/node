package node

import (
	"encoding/json"
	"errors"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"sync"
	"time"
)

var workProcess *workerProcess

type Context struct {
	srvConn *serverConnect
	MessageBaseI
}

func (c *Context) Reply(b []byte) error {
	if !c.srvConn.state {
		return errors.New(" wsasend: An exist ing connection was forcibly closed by the remote host")
	}
	if c.GetType() == MessageBaseType_Single || c.GetType() == MessageBaseType_SingleTranspond {
		return errors.New("message type reply not supported")
	}
	m := c.get()
	m.Data = b
	buf, err := m.Marshal()
	if err != nil {
		return err
	}

	if _, err = c.srvConn.conn.Write(jeans.Pack(buf)); err != nil {
		c.srvConn.state = false
		c.srvConn.close <- struct{}{}
		return err
	}
	return nil
}

func (c *Context) ReplyString(s string) error {
	return c.Reply([]byte(s))
}

func (c *Context) ReplyJson(a any) error {
	buf, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return c.Reply(buf)
}

func NewContext(srvConn *serverConnect, msg *MessageBase) *Context {
	var ctx = new(Context)
	ctx.srvConn = srvConn
	ctx.MessageBaseI = msg
	return ctx
}

type HandlerFunc func(ctx *Context)

type workerProcess struct {
	num           int
	handlerMap    *sync.Map
	tickHandle    *HandlerFunc
	noRouteHandle *HandlerFunc
	in            chan *Context
	close         chan struct{}
}

func (w *workerProcess) Start() {
	for i := 1; i <= w.num; i++ {
		go w.process(i)
	}
}

func (w *workerProcess) Stop() {
	close(w.close)
}

func (w *workerProcess) process(i int) {
	log.Printf("worker process start --- id[%v]\n", i)
	for {
		select {
		case ctx := <-w.in:
			//更新激活时间，防止连接被释放
			ctx.srvConn.activate = time.Now().UnixNano()
			if ctx.GetType() == MessageBaseType_Tick {
				fmt.Println("receive:", ctx.String())
				(*w.tickHandle)(ctx)
				continue
			}
			h, ok := w.handlerMap.Load(ctx.GetAPI())
			if !ok {
				log.Printf("worker process[%v] action api not exit : msgid [%v]\n", i, ctx.GetId())
				(*w.noRouteHandle)(ctx)
				continue
			}
			h.(HandlerFunc)(ctx)
		case <-w.close:
			log.Printf("worker process stop --- id[%v]\n", i)
			return
		}
	}
}

func newWorkerProcess(num int) *workerProcess {
	workProcess = new(workerProcess)
	workProcess.in = make(chan *Context, num)
	workProcess.close = make(chan struct{})
	workProcess.num = num
	return workProcess
}

func startWorkerProcess(num int, sm *sync.Map, noRouteHandle *HandlerFunc, tickHandle *HandlerFunc) {
	newWorkerProcess(num)
	workProcess.handlerMap = sm
	workProcess.noRouteHandle = noRouteHandle
	workProcess.tickHandle = tickHandle
	workProcess.Start()
}

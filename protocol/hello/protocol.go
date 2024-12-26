package hello

import (
	"errors"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"sync"
	"time"
)

type Event_Action uint8

func (h Event_Action) String() string {
	switch h {
	case Event_Action_Send_ACK:
		return "Hello_Action_Send_ACK"
	case Event_Action_Send_ASK:
		return "Hello_Action_Send_ASK"
	case Event_Action_Receive_ACK:
		return "Hello_Action_Receive_ACK"
	case Event_Action_Receive_ASK:
		return "Hello_Action_Receive_ASK"
	case Event_Action_TimeoutClose:
		return "Event_Action_TimeoutClose"
	case Event_Action_Error:
		return "Event_Action_Error"
	default:
		return "invalid action"
	}
}

const (
	Event_Action_Send_ACK Event_Action = iota
	Event_Action_Send_ASK
	Event_Action_Receive_ACK
	Event_Action_Receive_ASK
	Event_Action_TimeoutClose
	Event_Action_Error
)

const (
	Hello_ACK uint8 = iota
	Hello_ASK
)

type Protocol interface {
	iface.ConnectionLifecycle
	// Stop 停止
	Stop()
	// ReStart 重启
	ReStart()
	// SetEventCallback 产生的事件回调，在这里可以记录日志
	SetEventCallback(callback func(action Event_Action, val interface{}))
	ProtocolType() uint8
}

func NewHelloProtocol(protoType uint8, interval, timeout, timeoutClose time.Duration) Protocol {
	p := Hello{
		protoType:    protoType,
		nodeCache:    make(map[uint32]iface.Conn),
		interval:     interval,
		timeout:      timeout,
		timeoutClose: timeoutClose,
		exitChan:     make(chan struct{}, 1),
	}
	go p.Handle()
	return &p
}

type Hello struct {
	node.EmptyHandler
	protoType     uint8
	nodeCache     map[uint32]iface.Conn
	l             sync.Mutex
	interval      time.Duration
	timeout       time.Duration
	timeoutClose  time.Duration
	eventCallback func(action Event_Action, val interface{})
	exitChan      chan struct{}
}

func (h *Hello) OnConnect(conn iface.Conn) {
	h.l.Lock()
	h.nodeCache[conn.RemoteId()] = conn
	h.l.Unlock()
}

var actionErr = errors.New("OnCustomMessage receive \"action\" invalid")

func (h *Hello) OnMessage(ctx iface.Context) {
	var action uint8
	if len(ctx.Data()) != 1 {
		h.callEvent(Event_Action_Error, actionErr)
		return
	}
	action = ctx.Data()[0]
	switch action {
	case Hello_ACK:
		h.callEvent(Event_Action_Receive_ACK, ctx.SrcId())
	case Hello_ASK:
		_ = ctx.Conn().SendType(ctx.Type(), []byte{Hello_ACK})
		h.callEvent(Event_Action_Receive_ASK, ctx.DestId())
	}
}

func (h *Hello) OnClose(conn iface.Conn, err error) {
	h.l.Lock()
	delete(h.nodeCache, conn.RemoteId())
	h.l.Unlock()
}

func (h *Hello) ProtocolType() uint8 {
	return h.protoType
}

func (h *Hello) Handle() {
	tick := time.NewTicker(h.interval)
	go func() {
		<-h.exitChan
		tick.Stop()
	}()
	var diff time.Duration
	var data = []byte{Hello_ASK}
	for t := range tick.C {
		now := time.Duration(t.UnixNano())
		h.l.Lock()
		for _, conn := range h.nodeCache {
			diff = now - conn.Activate()
			if diff > h.timeoutClose {
				_ = conn.Close()
				h.callEvent(Event_Action_TimeoutClose, conn.RemoteId())
			} else if diff > h.timeout {
				_ = conn.SendType(h.protoType, data)
				h.callEvent(Event_Action_Send_ASK, conn.RemoteId())
			}
		}
		h.l.Unlock()
	}
}

func (h *Hello) callEvent(action Event_Action, val interface{}) {
	if h.eventCallback != nil {
		h.eventCallback(action, val)
	}
}

func (h *Hello) ReStart() {
	h.exitChan = make(chan struct{}, 1)
	go h.Handle()
}

func (h *Hello) Stop() {
	close(h.exitChan)
}

func (h *Hello) SetEventCallback(callback func(action Event_Action, val interface{})) {
	h.eventCallback = callback
}

package node

import (
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

var _rnd *rand.Rand

func init() {
	_rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// 1024-49151
func getPort() string {
	return strconv.Itoa(_rnd.Intn(49152-1024) + 1024)
}

// 测试用：原子计数器
type Counter struct {
	start time.Time
	end   time.Time
	w     sync.WaitGroup
	num   int
}

func NewCounter() *Counter {
	return new(Counter)
}

func (c *Counter) AsyncRun(num int, f func()) {
	c.w.Add(num)
	c.num = num
	c.start = time.Now()
	for i := 0; i < num; i++ {
		go func() {
			defer c.w.Done()
			f()
		}()
	}
	c.w.Wait()
	c.end = time.Now()
}

func (c *Counter) Run(num int, f func()) {
	c.num = num
	c.start = time.Now()
	for i := 0; i < num; i++ {
		f()
	}
	c.end = time.Now()
}

func (c *Counter) String() string {
	t := c.end.Sub(c.start)
	return fmt.Sprintf("run info: num [%v] duration [%v] avg [%v]", c.num, t, time.Duration(t.Nanoseconds()/int64(c.num)).String())
}

func (c *Counter) Debug() {
	fmt.Println(c.String())
}

func readMessage(conn *net.TCPConn) (*message, error) {
	buf, err := jeans.Unpack(conn)
	if err != nil {
		return nil, err
	}
	return newMsgWithUnmarshalV2(buf), nil
}

func writeMsg(conn *net.TCPConn, m *message) error {
	_, err := conn.Write(jeans.Pack(m.marshalV2()))
	return err
}

func write(conn *net.TCPConn, data []byte) error {
	_, err := conn.Write(jeans.Pack(data))
	return err
}

func parseAddress(protocol string, addr ...string) ([]*net.TCPAddr, error) {
	a := make([]*net.TCPAddr, 0, len(addr))
	for _, item := range addr {
		tmp, err := net.ResolveTCPAddr(protocol, item)
		if err != nil {
			return nil, err
		}
		a = append(a, tmp)
	}
	return a, nil
}

func srvConnHandle(router *Handler, forward func(msg *message) error) func(i interface{}) {
	return func(i interface{}) {
		ctx := i.(*Context)
		switch ctx._type {
		case MsgType_ServerRespSuccess, MsgType_ServerRespFail:
			v, ok := ctx.setRespChan(ctx.id)
			if !ok {
				log.Println("Receive timeout message or push message:", ctx.message.String())
				break
			}
			v.(chan *message) <- ctx.message
		case MsgType_Req, MsgType_Send:
			handler, ok := router.handle[ctx.message.API]
			if !ok {
				switch ctx._type {
				case MsgType_Send:
					//不通知返回
				case MsgType_Req:
					ctx._type = MsgType_RespFail
					ctx.Data = []byte(ErrNoApi.Error())
					_ = ctx.write(ctx.message)
				}
				return
			}
			data, err := handler(ctx.srcId, ctx.Data)
			if ctx._type == MsgType_Send {
				return
			}
			if err != nil {
				ctx._type = MsgType_RespFail
				ctx.Data = []byte(err.Error())
			} else {
				ctx.Data = data
				ctx._type = MsgType_RespSuccess
			}
			_ = ctx.write(ctx.message)
		case MsgType_ReqForward, MsgType_RespForwardSuccess, MsgType_RespForwardFail:
			err := forward(ctx.message)
			if err != nil && ctx._type == MsgType_ReqForward {
				ctx._type = MsgType_RespForwardFail
				ctx.Data = []byte(err.Error())
				_ = ctx.write(ctx.message)
			}
		case MsgType_Tick:
			ctx._type = MsgType_TickResp
			_ = ctx.write(ctx.message)
		default:
			fmt.Println("default handle:", ctx.message.String())
		}
	}
}

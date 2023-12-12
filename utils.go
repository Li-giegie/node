package node

import (
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"
)

var _rnd *rand.Rand

type AuthenticationFunc func(id uint64, data []byte) (ok bool, reply []byte)

func init() {
	_rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// 1024-49151
func getPort() string {
	return strconv.Itoa(_rnd.Intn(49152-1024) + 1024)
}

func readMessage(conn *net.TCPConn) (*message, error) {
	buf, err := jeans.Unpack(conn)
	if err != nil {
		return nil, err
	}
	return newMsgWithUnmarshalV2(buf), nil
}

func writeMsg(conn *net.TCPConn, m *message) error {
	buf := m.marshalV2()
	m.recycle()
	_, err := conn.Write(jeans.Pack(buf))
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
		ctx := i.(*nodeContext)
		switch ctx.typ {
		case msgType_RespSuccess, msgType_RespFail:
			v, ok := ctx.setRespChan(ctx.id)
			if !ok {
				log.Println("Receive timeout message or push message:", ctx.message.String())
				break
			}
			v.(chan *message) <- ctx.message
		case msgType_Req, msgType_Send:
			handler, ok := router.handle[ctx.message.api]
			if !ok {
				switch ctx.typ {
				case msgType_Send:
					//不通知返回
				case msgType_Req:
					ctx.typ = msgType_RespFail
					ctx.data = []byte(ErrNoApi.Error())
					_ = ctx.write(ctx.message)
				}
				return
			}
			data, err := handler(ctx.srcId, ctx.data)
			if ctx.typ == msgType_Send {
				return
			}
			if err != nil {
				ctx.typ = msgType_RespFail
				ctx.data = []byte(err.Error())
			} else {
				ctx.data = data
				ctx.typ = msgType_RespSuccess
			}
			_ = ctx.write(ctx.message)
		case msgType_Forward, msgType_ForwardSuccess, msgType_ForwardFail:
			err := forward(ctx.message)
			if err != nil && ctx.typ == msgType_Forward {
				ctx.typ = msgType_ForwardFail
				ctx.data = []byte(err.Error())
				_ = ctx.write(ctx.message)
			}
		case msgType_Tick:
			ctx.typ = msgType_TickResp
			_ = ctx.write(ctx.message)
		default:
			fmt.Println("default handle:", ctx.message.String())
		}
	}
}

package node

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"strings"
	"testing"
	"time"
)

type ReqScene struct {
}

func (ReqScene) Hello() []byte {
	return []byte("hello scene1")
}

func (ReqScene) Id() uint32 {
	return 1
}

func (ReqScene) Handler() HandleFunc {
	return func(id uint64, data []byte) (out []byte, err error) {
		if string(data) == "wait" {
			time.Sleep(time.Second * 3)
		}
		switch string(data) {
		case "wait":
			time.Sleep(time.Second * 3)
			return []byte("ReqScene success"), nil
		case "send":
			return nil, nil
		case "send print":
			fmt.Println(string(data))
			return nil, nil
		default:
			return []byte("ReqScene success"), nil
		}
	}
}

const (
	sendApi    = 1
	reqApi     = 2
	forwardApi = 3

	permit = "permit"
	deny   = "deny"

	performance_ReqTestApi = 4
)

func TestNodeServer(t *testing.T) {
	srv, err := NewServer(DEFAULT_ServerAddress, WithSrvGoroutine(100, 200), WithSrvId(DEFAULT_ServerID))
	if err != nil {
		t.Error(err)
	}
	defer srv.Shutdown()

	//测试认证 根据消息决定是否通过
	srv.AuthenticationFunc = func(id uint64, data []byte) (ok bool, reply []byte) {
		fmt.Println("auth handle:", id, string(data), len(data))
		switch string(data) {
		case permit:
			return true, []byte(permit)
		case deny:
			return false, []byte(deny)
		default:
			return false, []byte("Invalid Format")
		}
	}

	//添加客户端只发送不响应的路由，继续转发、请求其他客户端
	srv.HandleFunc(sendApi, func(id uint64, data []byte) (out []byte, err error) {
		log.Printf("server send scene id %d data %s\n", id, data)
		if len(data) == 0 {
			return nil, nil
		}
		var respBuf []byte
		switch data[0] {
		case MsgType_ReqForward: //转发send
			err = srv.Send(binary.LittleEndian.Uint64(data[1:9]), binary.LittleEndian.Uint32(data[9:12]), data[12:])
		case MsgType_Req:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			respBuf, err = srv.Request(ctx, binary.LittleEndian.Uint64(data[1:9]), binary.LittleEndian.Uint32(data[9:12]), data[12:])
		default:
			log.Println("send err ", string(data))
			return nil, err
		}
		if err != nil {
			if data[0] == MsgType_ReqForward {
				fmt.Println(string(respBuf))
			}
			fmt.Println(err)
			return nil, err
		}
		return nil, nil
	})

	srv.HandleFunc(reqApi, func(id uint64, data []byte) (out []byte, err error) {
		fmt.Println(data, strings.Contains(string(data), "@"))
		log.Printf("server request scene id %d data %s\n", id, data)
		if len(data) == 0 {
			return nil, nil
		}
		var _t uint8
		var _id uint64
		var _api uint32
		var _data []byte
		err = jeans.Decode(data, &_t, &_id, &_api, &_data)
		if err != nil {
			log.Println(err)
			return append([]byte("server handle decode fail reply src data: "), data...), nil
		}
		var respBuf []byte
		switch _t {
		case 0:
			return respBuf, errors.New("invalid Format")
		case MsgType_Send: //转发send
			err = srv.Send(binary.LittleEndian.Uint64(data[1:9]), binary.LittleEndian.Uint32(data[9:12]), data[12:])
		case MsgType_Req:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			respBuf, err = srv.Request(ctx, _id, _api, _data)
		case 100:
			time.Sleep(time.Second * 3)
			respBuf = []byte("server handle timeout reply")
		default:
			respBuf = append([]byte("server handle success reply src data: "), data...)
		}
		return respBuf, err
	})

	srv.HandleFunc(performance_ReqTestApi, func(id uint64, data []byte) (out []byte, err error) {
		return data, nil
	})

	if err = srv.ListenAndServer(); err != nil {
		fmt.Println(err)
	}
}

package node

import (
	"context"
	"errors"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"strings"
	"testing"
	"time"
)

const (
	sendApi    = 1
	reqApi     = 2
	forwardApi = 3

	permit = "permit"
	deny   = "deny"

	performance_ReqTestApi = 4
)

func TestNodeServer(t *testing.T) {
	srv := NewServer(DEFAULT_ServerAddress,
		WithSrvGoroutine(100, 200),
		WithSrvId(DEFAULT_ServerID),
		WithSrvAuthentication(func(id uint64, data []byte) (ok bool, reply []byte) {
			fmt.Println("auth handle:", id, string(data), len(data))
			switch string(data) {
			case permit:
				return true, []byte(permit)
			case deny:
				return false, []byte(deny)
			default:
				return false, []byte("Invalid Format")
			}
		},
		),
	)

	defer srv.Shutdown()
	//添加客户端只发送不响应的路由，继续转发、请求其他客户端
	srv.HandleFunc(sendApi, func(id uint64, data []byte) (out []byte, err error) {
		log.Printf("server send scene id %d data %s\n", id, data)

		switch data[0] {
		case 0:

		default:
			var _t uint8
			var _id uint64
			var _api uint32
			var _data []byte
			if err = jeans.Decode(data, &_t, &_id, &_api, &_data); err != nil {
				log.Println("decode fail ", err)
				return nil, err
			}
			switch _t {
			case msgType_Send: //转发send
				srv.FindConn(id)
				conn, ok := srv.FindConn(_id)
				if !ok {
					return nil, err
				}
				_ = conn.Send(_api, data)
			case msgType_Req:
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				conn, ok := srv.FindConn(_id)
				if !ok {
					return nil, err
				}
				_, err = conn.Request(ctx, _api, data)
			}
		}
		if err != nil {
			log.Println(err)
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
		case msgType_Send: //转发send
			conn, ok := srv.FindConn(_id)
			if !ok {
				return nil, err
			}
			err = conn.Send(_api, data)

		case msgType_Req:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			conn, ok := srv.FindConn(_id)
			if !ok {
				return nil, err
			}
			respBuf, err = conn.Request(ctx, _api, data)
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

	if err := srv.ListenAndServer(); err != nil {
		fmt.Println(err)
	}
}

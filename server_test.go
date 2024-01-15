package node

import (
	"errors"
	"fmt"
	"log"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	srv := NewServer(DEFAULT_ServerAddress,
		WithSrvConnTimeout(time.Second*5),
		WithSrvAuthentication(func(id uint64, data []byte) (reply []byte, err error) {
			log.Println("auth: ", id, string(data))
			if len(data) == 0 {
				return []byte("data "), errors.New("deny pass")
			}
			return nil, nil
		},
		))
	srv.HandleFunc(1, func(ctx *Context) {
		//if err := ctx.Reply([]byte("1")); err != nil {
		//	fmt.Println(err)
		//}
	})
	srv.HandleFunc(2, func(ctx *Context) {
		log.Println("handle 2: ", ctx.SrcId(), string(ctx.Data()))
		if err := ctx.Reply([]byte("ok---2")); err != nil {
			log.Println("reply err: ", err)
		}
	})
	srv.HandleFunc(3, func(ctx *Context) {
		log.Println("handle 3: ", ctx.SrcId(), string(ctx.Data()))
		if err := ctx.ReplyErr(errors.New("err: error test"), []byte("ok---2")); err != nil {
			log.Println("reply err: ", err)
		}
	})
	if err := srv.ListenAndServer(true); err != nil {
		fmt.Println(err)
	}
}

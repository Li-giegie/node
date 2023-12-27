package node

import (
	"fmt"
	"log"
	"testing"
)

func TestServer(t *testing.T) {
	srv := NewServer(DEFAULT_ServerAddress, WithSrvAuthentication(func(id uint64, data []byte) (ok bool, reply []byte) {
		log.Println(id, string(data))
		return true, nil
	}))
	srv.HandleFunc(1, func(ctx *Context) {
		//if err := ctx.Reply([]byte("1")); err != nil {
		//	fmt.Println(err)
		//}
	})
	srv.HandleFunc(2, func(ctx *Context) {
		log.Println("handle 2: ", ctx.GetSrcId(), string(ctx.GetData()))
		if err := ctx.Reply([]byte("ok---2")); err != nil {
			log.Println("reply err: ", err)
		}
	})
	if err := srv.ListenAndServer(); err != nil {
		fmt.Println(err)
	}
}

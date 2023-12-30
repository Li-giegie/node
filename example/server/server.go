package main

import (
	"flag"
	"github.com/Li-giegie/node"
	"log"
	"time"
)

func Server(addr string, id uint64) {
	srv := node.NewServer(addr,
		node.WithSrvId(id),
		node.WithSrvConnTimeout(time.Second*10),
		node.WithSrvConnectionEnableFunc(func(conn node.ISrvConn) {
			log.Println("authentication success new connect ---", conn.GetId())
		}),
		node.WithSrvAuthentication(func(id uint64, data []byte) (ok bool, reply []byte) {
			log.Println("authentication: ", id, string(data))
			return true, nil
		}),
	)
	srv.HandleFunc(1, func(ctx *node.Context) {
		log.Println("receive msg with handle 1: ", ctx.String())
	})
	srv.HandleFunc(2, func(ctx *node.Context) {
		log.Println("receive msg with handle 2: ", ctx.String())
		_ = ctx.Reply(append([]byte("receive success"), ctx.GetData()...))
	})
	defer srv.Shutdown()
	if err := srv.ListenAndServer(); err != nil {
		log.Fatalln(err)
	}
}

var lAddr = flag.String("laddr", node.DEFAULT_ServerAddress, "local address")
var id = flag.Uint64("id", node.DEFAULT_ServerID, "id")

func main() {
	flag.Parse()
	Server(*lAddr, *id)
}

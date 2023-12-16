package main

import (
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"log"
	"time"
)

const (
	SendApi = 1
	ReqApi  = 2
)

func Server(addr string, id uint64) {
	srv := node.NewServer(addr,
		node.WithSrvId(id),
		node.WithSrvConnTimeout(time.Second*10),
		node.WithSrvConnectionEnableFunc(func(conn node.Conn) {
			log.Println("authentication success new connect ---", conn.Id())
		}),
		node.WithSrvAuthentication(func(id uint64, data []byte) (ok bool, reply []byte) {
			log.Println("authentication: ", id, string(data))
			return true, nil
		}),
	)
	srv.HandleFunc(SendApi, func(id uint64, data []byte) (out []byte, err error) {
		fmt.Println("send api test: ", id, string(data))
		return nil, nil
	})
	srv.HandleFunc(ReqApi, func(id uint64, data []byte) (out []byte, err error) {
		fmt.Println("req api test:", id, string(data))
		return nil, nil
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

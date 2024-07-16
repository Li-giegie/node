package main

import (
	"flag"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/protocol"
	"log"
	"net"
	"time"
)

var id = flag.Uint("id", 0, "id")
var laddr = flag.String("laddr", "", "local address")
var raddr = flag.String("raddr", "", "remote address")
var key = flag.String("key", "hello", "auth key")

func main() {
	flag.Parse()
	externalDomainNode, err := node.DialExternalDomainNode("tcp", *raddr, func(conn net.Conn) (remoteId uint16, err error) {
		return protocol.NewClientAuthProtocol(uint16(*id), *key, time.Second*5).Init(conn)
	})
	if err != nil {
		log.Fatalln(err)
	}
	srv := NewServerHandle(uint16(*id), *laddr, "hello", time.Second*6)
	if err = srv.Listen(); err != nil {
		log.Fatalln(err)
	}
	defer srv.Close()
	go func() {
		if err = srv.Bind(externalDomainNode); err != nil {
			log.Println(err)
			return
		}
	}()
	if err = srv.Serve(); err != nil {
		log.Fatalln(err)
	}
}

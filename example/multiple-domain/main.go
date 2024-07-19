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
var key = flag.String("key", "hello", "auth key")
var raddr = flag.String("raddr", "", "remote address")
var enableBind = flag.Bool("enablebind", false, "enable bind node -raddr")

func main() {
	flag.Parse()
	srv := NewServerHandle(uint16(*id), *laddr, "hello", time.Second*6)
	if err := srv.Listen(); err != nil {
		log.Fatalln(err)
	}
	log.Printf("server [%d] start success\n", *id)
	stop := make(chan error, 1)
	go func() {
		stop <- srv.Serve()
	}()
	if *enableBind {
		externalDomainNode, err := node.DialExternalDomainNode("tcp", *raddr, func(conn net.Conn) (remoteId uint16, err error) {
			return protocol.NewClientAuthProtocol(uint16(*id), *key, time.Second*5).Init(conn)
		})
		if err != nil {
			stop <- err
		}
		log.Println("external domain node start success")
		go func() {
			stop <- srv.Bind(externalDomainNode)
		}()
	}
	err := <-stop
	log.Printf("server [%d] exit %v\n", *id, err)

}

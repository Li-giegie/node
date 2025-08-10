package main

import (
	"bufio"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/reply"
	"github.com/Li-giegie/node/pkg/server"
	"log"
	"net"
	"os"
)

func main() {
	srv := node.NewServerOption(1)
	server.OnAccept(func(conn net.Conn) (next bool) {
		log.Println("OnAccept", conn.RemoteAddr())
		return true
	})
	server.OnConnect(func(conn *conn.Conn) (next bool) {
		log.Println("OnConnect", conn.RemoteAddr())
		return true
	})
	server.OnMessage(func(r *reply.Reply, m *message.Message) (next bool) {
		log.Println("OnMessage", m.String())
		r.String(message.StateCode_Success, "pong")
		return true
	})
	server.OnClose(func(conn *conn.Conn, err error) (next bool) {
		log.Println("OnClose", conn.RemoteAddr())
		return true
	})
	go func() {
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			if len(s.Bytes()) == 0 {
				continue
			}
			srv.RangeConn(func(conn *conn.Conn) bool {
				conn.Send(s.Bytes())
				return true
			})
		}
	}()
	log.Println("listen: 7891")
	err := srv.ListenAndServe(":7891", nil)
	if err != nil {
		log.Fatal(err)
	}

}

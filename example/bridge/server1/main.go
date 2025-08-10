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
	"time"
)

func main() {
	srv := node.NewServerOption(2)
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
	}()
	go func() {
		defer srv.Close()
		log.Println("listen: 7892")
		err := srv.ListenAndServe(":7892", nil)
		if err != nil {
			log.Fatal(err)
		}
	}()
	time.Sleep(time.Second)
	conn, err := net.Dial("tcp", "127.0.0.1:7891")
	if err != nil {
		log.Println(err)
		return
	}
	if err = srv.Bridge(conn, 1, nil); err != nil {
		log.Println(err)
		return
	}
	log.Println("bridge ok")
	for {
	}
}

package main

import (
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"log"
	"os"
	"time"
)

type logWriter struct {
	f *os.File
}

func newLogWriter(name string) *logWriter {
	f, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	return &logWriter{f: f}
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	fmt.Println(string(p))
	return w.f.Write(p)
}

func Server(addr string, id uint64) {
	log.SetOutput(newLogWriter("./server-debug.log"))
	srv := node.NewServer(addr,
		node.WithSrvId(id),
		node.WithSrvConnTimeout(time.Second*10),
		node.WithSrvAuthentication(func(id uint64, data []byte) (reply []byte, err error) {
			log.Println("authentication: ", id, string(data))
			return nil, nil
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
	if err := srv.ListenAndServer(true); err != nil {
		log.Fatalln(err)
	}
}

var lAddr = flag.String("laddr", node.DEFAULT_ServerAddress, "local address")
var id = flag.Uint64("id", node.DEFAULT_ServerID, "id")

func main() {
	flag.Parse()
	Server(*lAddr, *id)
}

package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"
)

var echoConn iface.Conn

func Dial() {
	netConn, err := net.Dial("tcp", "0.0.0.0:8000")
	if err != nil {
		panic(err)
	}
	stopC := make(chan struct{})
	c := node.NewClient(8001, &node.Identity{Id: 8000, Key: []byte("hello"), Timeout: time.Second * 6}, nil)
	c.AddOnMessage(func(ctx iface.Context) {
		fmt.Println(ctx.String())
		ctx.Reply(ctx.Data())
	})
	c.AddOnClose(func(conn iface.Conn, err error) {
		stopC <- struct{}{}
	})
	conn, err := c.Start(netConn)
	if err != nil {
		panic(err)
	}
	echoConn = conn
	time.Sleep(time.Second)
}

var once = sync.Once{}

func BenchmarkEchoRequest(b *testing.B) {
	once.Do(func() {
		Dial()
		b.ResetTimer()
	})
	ctx := context.Background()
	wg := sync.WaitGroup{}
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			_, err := echoConn.Request(ctx, []byte("hello"))
			if err != nil {
				b.Error(err)
				return
			}
			wg.Done()
		}()
	}
	wg.Wait()
	//1000000              1572 ns/op             333 B/op         7 allocs/op
	//93460             13591 ns/op             682 B/op         8 allocs/op
	//fmt.Println("write Trace", nodeNet.WriteTrace)
	//net.PrintTrace()
}

func BenchmarkEchoRequestGo(b *testing.B) {
	once.Do(func() {
		Dial()
		b.ResetTimer()
	})
	ctx := context.Background()
	w := sync.WaitGroup{}
	for i := 0; i < b.N; i++ {
		w.Add(1)
		go func() {
			res, err := echoConn.Request(ctx, []byte("hello"))
			if err != nil {
				fmt.Println(err, res)
			}
			w.Done()
		}()
	}
	w.Wait()
	//fmt.Println()
	//net.PrintTrace()
}

func TestEchoClient(t *testing.T) {
	Dial()
	ctx := context.Background()
	t1 := time.Now()
	w := sync.WaitGroup{}
	for i := 0; i < 1000000; i++ {
		w.Add(1)
		go func() {
			res, err := echoConn.Request(ctx, []byte(strconv.Itoa(i)))
			if err != nil {
				fmt.Println(err, res)
				w.Done()
				return
			}
			w.Done()
		}()
	}
	w.Wait()
	//965926
	fmt.Println(time.Since(t1))
}

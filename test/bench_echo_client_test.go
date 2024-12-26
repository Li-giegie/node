package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"strconv"
	"sync"
	"testing"
	"time"
)

var echoConn iface.Client

func Dial() {
	c := node.NewClient(8001, &node.Identity{Id: 8000, Key: []byte("hello"), AuthTimeout: time.Second * 6}, nil)
	c.OnMessage(func(ctx iface.Context) {
		fmt.Println(ctx.String())
		ctx.Response(200, ctx.Data())
	})
	err := c.Connect("0.0.0.0:8000")
	if err != nil {
		panic(err)
	}
	echoConn = c
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
			_, _, err := echoConn.Request(ctx, []byte("hello"))
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
			_, res, err := echoConn.Request(ctx, []byte("hello"))
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
			_, res, err := echoConn.Request(ctx, []byte(strconv.Itoa(i)))
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

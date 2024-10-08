package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"strconv"
	"sync"
	"testing"
	"time"
)

var echoConn node.Conn

func Dial() {
	conn, err := node.DialTCP(
		"0.0.0.0:8888",
		&node.Identity{
			Id:            0xffffff,
			AccessKey:     []byte("echo"),
			AccessTimeout: time.Second * 3,
		},
		&Echo{},
	)
	if err != nil {
		panic(err)
	}
	echoConn = conn

}

var once = sync.Once{}

func BenchmarkEchoRequest(b *testing.B) {
	once.Do(func() {
		Dial()
		b.ResetTimer()
	})
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		_, err := echoConn.Request(ctx, []byte("hello"))
		if err != nil {
			b.Error(err)
			return
		}
	}
	//fmt.Println()
	//common.PrintTrace()
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
			_, err := echoConn.Request(ctx, []byte("hello"))
			if err != nil {
				b.Error(err)
				w.Done()
				return
			}
			w.Done()
		}()
	}
	w.Wait()
	//fmt.Println()
	//common.PrintTrace()
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
	fmt.Println(time.Since(t1))
}

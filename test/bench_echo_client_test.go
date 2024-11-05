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

var echoConn iface.Client

func Dial() {
	conn, err := net.Dial("tcp", "0.0.0.0:8888")
	if err != nil {
		panic(err)
	}
	echoConn = node.NewClient(conn, &node.CliConf{
		ReaderBufSize:   4096,
		WriterBufSize:   4096,
		WriterQueueSize: 1024,
		MaxMsgLen:       0xffffff,
		ClientIdentity: &node.ClientIdentity{
			Id:            1234,
			RemoteAuthKey: []byte("hello"),
			Timeout:       time.Second,
		},
	})
	if err = echoConn.Start(); err != nil {
		panic(err)
	}
	time.Sleep(time.Second)
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
	fmt.Println(time.Since(t1))
}

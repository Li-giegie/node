package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"log"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestClientSendMsg_1s(t *testing.T) {
	client, err := node.NewClient("127.0.0.1:2023")
	if err != nil {
		t.Error(err)
		return
	}
	conn, err := client.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	var c node.Counter
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	for {
		c.AddRequestNum()
		select {
		case <-ctx.Done():
			c.Debug()
			return
		default:
			err = conn.Send(1, []byte("hello word"))
			if err != nil {
				c.Debug()
				return
			}
			c.AddReplyNum()
		}
	}
}

func TestClientRequestMsg_1s(t *testing.T) {
	client, err := node.NewClient("127.0.0.1:2023")
	if err != nil {
		t.Error(err)
		return
	}
	conn, err := client.Connect()
	if err != nil {
		t.Error(err)
		return
	}

	var count node.Counter
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	for {
		count.AddRequestNum()
		select {
		case <-ctx.Done():
			count.Debug()
			return
		default:
			go func() {
				reply, err := conn.Request(ctx, 1, []byte("request"))
				if err != nil {
					count.Debug()
					fmt.Println(reply.String())
					return
				}
				count.AddReplyNum()
			}()
		}
	}
}

func TestClientRequestMsgNum(t *testing.T) {
	client, err := node.NewClient("127.0.0.1:2023")
	if err != nil {
		t.Error(err)
		return
	}
	conn, err := client.Connect()
	if err != nil {
		t.Error(err)
		return
	}

	var count = node.NewCounter()

	ctx, _ := context.WithTimeout(context.Background(), time.Second)

	for i := 0; i < 100000; i++ {

		reply, err := conn.Request(ctx, 1, []byte("request"))
		if err != nil {
			count.Debug()
			fmt.Println(reply.String())
			return
		}
		count.AddReplyNum()
	}
	count.Debug()
}

func TestClient100WSend(t *testing.T) {
	client, err := node.NewClient("127.0.0.1:2023")
	if err != nil {
		t.Error(err)
		return
	}
	conn, err := client.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	n := 100000
	t1 := time.Now()
	for i := 0; i < n; i++ {
		err := conn.Send(1, []byte("hello server !"))
		if err != nil {
			t.Error(err)
			return
		}
	}
	fmt.Printf("执行次数 [%v] 耗时 [%v]", n, time.Since(t1))
}

// go test -bench=BenchmarkBaseTypeToBytes$   -benchtime=3s .\ -cpuprofile="BenchmarkBaseTypeToBytes_CPUV1.out"
func Test_Cli(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:2023")
	if err != nil {
		log.Fatalln(err)
	}

	t1 := time.Now()
	for i := 0; i < 100000; i++ {
		_, err = conn.Write([]byte(strconv.Itoa(i)))
		if err != nil {
			fmt.Println("退出")
			return
		}
	}

	conn.Close()
	fmt.Println(time.Since(t1))
}

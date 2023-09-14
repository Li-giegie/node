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

func getMustConn() *node.ClientConnect {
	testClient, err := node.NewClient("127.0.0.1:2023")
	if err != nil {
		log.Fatalln(err)
	}
	testClientConn, err := testClient.Connect()
	if err != nil {
		log.Fatalln(err)
	}
	return testClientConn
}

func TestClientSend(t *testing.T) {
	err := getMustConn().Send(1, []byte("hello word"))
	if err != nil {
		t.Error(err)
		return
	}
}

func TestNodeClientRequest(t *testing.T) {
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
	defer conn.Close()
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	reply, err := conn.Request(ctx, 2, []byte("request"))
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(reply.String())
}

func TestClientTick(t *testing.T) {
	cli, err := node.NewClient(node.DEFAULT_ServerAddress)
	if err != nil {
		t.Error(err)
		return
	}
	cli.DetectionKeepAlive = time.Second
	cli.KeepAlive = time.Second * 3
	conn, err := cli.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()

	for i := 0; i < 20; i++ {
		time.Sleep(time.Second)
	}
}

func TestClientSend_100W(t *testing.T) {
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
	var c = node.NewCounter()
	for i := 0; i < 1000000; i++ {
		err = conn.Send(1, []byte("hello server !"))
		if err != nil {
			t.Error(err)
			break
		}
		c.AddReplyNum()
	}
	c.Debug()
}

// 耗时 11.7215171s 效率 request num:[0],reply num:[100000]
// 耗时 11.7950453s 效率 request num:[0],reply num:[100000]
// 耗时 35.3810592s 效率 request num:[0],reply num:[300000]
func TestClientRequest_10W(t *testing.T) {
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
	ctx, _ := context.WithTimeout(context.Background(), time.Second*40)
	var c = node.NewCounter()
	for i := 0; i < 1000000; i++ {
		_, err = conn.Request(ctx, 2, []byte("hello server !"))
		if err != nil {
			t.Error(err)
			break
		}
		c.AddReplyNum()
	}
	c.Debug()

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

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
	testClient := node.NewClient(node.DEFAULT_ClientID, node.DEFAULT_ServerAddress)
	testClientConn, err := testClient.Connect(nil)
	if err != nil {
		log.Fatalln(err)
	}
	return testClientConn
}

func TestClientSend(t *testing.T) {
	fmt.Println(getMustConn().Send(1, []byte("hello")))
	return
	testClient := node.NewClient(node.DEFAULT_ClientID, node.DEFAULT_ServerAddress)
	conn, err := testClient.Connect(nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()
	fmt.Println(conn.Send(1, []byte("hello")))
}

func TestNodeClientRequest(t *testing.T) {
	client := node.NewClient(node.DEFAULT_ClientID, node.DEFAULT_ServerAddress)
	client.AddRoute(1, node.HandlerFunc(func(ctx *node.Context) {
		fmt.Println(ctx.String())
	}))
	conn, err := client.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()
	ctx, _ := context.WithTimeout(context.Background(), time.Second*20)
	reply, err := conn.Request(ctx, 2, []byte("request 2"))
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println("reply 2 ", reply.String())

	time.Sleep(time.Second * 15)

}

func TestClientTick(t *testing.T) {
	cli := node.NewClient(node.DEFAULT_ClientID, node.DEFAULT_ServerAddress)
	cli.DetectionKeepAlive = time.Second
	cli.KeepAlive = time.Second * 3
	conn, err := cli.Connect([]byte{})
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
	client := node.NewClient(node.DEFAULT_ClientID, node.DEFAULT_ServerAddress)
	conn, err := client.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()
	var c = node.NewCounter()
	for i := 0; i < 1000000; i++ {
		c.AddSend()
		err = conn.Send(1, []byte("hello server !"))
		if err != nil {
			t.Error(err)
			break
		}
	}
	c.Debug()
}

func TestClientRequest_100W(t *testing.T) {
	client := node.NewClient(node.DEFAULT_ClientID, node.DEFAULT_ServerAddress)
	conn, err := client.Connect(nil)
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
		c.AddSend()
	}
	c.Debug()

}

func TestClientForwardClientServer(t *testing.T) {

}

func TestNodeClientForwardRequest(t *testing.T) {
	client := node.NewClient(node.DEFAULT_ClientID, node.DEFAULT_ServerAddress)
	client.AddRoute(1, func(ctx *node.Context) {
		fmt.Println("收到转发消息：", node.NewMessageForwardWithUnmarshal(ctx.GetData()).String())
		fmt.Println("回复：", ctx.Write([]byte("forward success!")))
	})
	conn, err := client.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()
	ctx, _ := context.WithTimeout(context.Background(), time.Second*20)
	msg, err := conn.RequestForward(ctx, node.DEFAULT_ClientID, node.DEFAULT_ClientID, 1, []byte("forward"))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("forward resp :", err, msg.String())
	m := new(node.MessageForward)
	m.Unmarshal(msg.Data)
	fmt.Println(*m)
}

func TestSingleForward(t *testing.T) {
	fmt.Println(getMustConn().SingleForward(node.DEFAULT_ClientID, node.DEFAULT_ClientID, 1, []byte("asd")))
}

// go test -bench=BenchmarkBaseTypeToBytes$   -benchtime=3s .\ -cpuprofile="BenchmarkBaseTypeToBytes_CPUV1.out"
func Test_Cli(t *testing.T) {
	conn, err := net.Dial("tcp", node.DEFAULT_ServerAddress)
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

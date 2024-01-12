package node

import (
	"fmt"
	utils "github.com/Li-giegie/go-utils"
	"log"
	"math"
	"os"
	"testing"
	"time"
)

func TestClient_Auth(t *testing.T) {
	cli := NewClient(DEFAULT_ServerAddress, WithClientKeepAlive(time.Second*5))
	reply, err := cli.Connect(DEFAULT_ServerID, []byte{1})
	if err != nil {
		t.Error(err, string(reply))
		return
	}
	fmt.Println(string(reply))
	fmt.Println(cli.Run())
}

func TestClient_Registration(t *testing.T) {
	cli := NewClient(DEFAULT_ServerAddress,
		WithClientId(3790),
		WithClientLocalIpAddr("127.0.0.1:6667"),
		WithClientKeepAlive(time.Second*2))
	authReply, err := cli.Connect(DEFAULT_ServerID, []byte("hello"))
	if err != nil {
		t.Error(err, string(authReply))
		return
	}
	cli.HandleFunc(100, func(ctx *Context) {
		fmt.Println("send:", ctx.String())
	})
	cli.HandleFunc(200, func(ctx *Context) {
		fmt.Println("send 2:", ctx.String())
		ctx.Reply([]byte("receive success"))
	})
	if badApi, err := cli.Registration(); err != nil {
		fmt.Println("api: ", badApi)
		t.Error(err)
		return
	}
	fmt.Println("aa")
	if err := cli.Run(); err != nil {
		t.Error(err)
	}
}

func TestClientSend(t *testing.T) {
	cli := NewClient(DEFAULT_ServerAddress, WithClientId(6655), WithClientLocalIpAddr("0.0.0.0:6655"))
	authReply, err := cli.Connect(DEFAULT_ServerID, []byte("hello"))
	if err != nil {
		t.Error(err, string(authReply))
		return
	}

	var i int
	for {
		reply, err := cli.Request(time.Second*3, 200, []byte("hello"))
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Printf("reply: %s\n", reply)
		time.Sleep(time.Second)
		continue
		utils.AsyncRun(100000, func() {
			if err = cli.Send(100, []byte("send 1")); err != nil {
				log.Println(err)
			}
		}).Debug()
		i += 10000
		if i >= math.MaxInt {
			break
		}
	}
	//result: sum duration: [2.7612166s], avg time: [27.612µs], num: [100000], mode: [AsyncRun]
	utils.AsyncRun(100000, func() {
		if err = cli.Send(100, []byte("send 1")); err != nil {
			log.Println(err)
		}
	}).Debug()
}

func TestClientRequest(t *testing.T) {
	cli := NewClient("39.101.193.248:8088", WithClientLocalIpAddr("0.0.0.0:7788"))
	authReply, err := cli.Connect(DEFAULT_ServerID, []byte("hello"))
	if err != nil {
		t.Error("auth err: ", err, string(authReply))
		return
	}
	reply, err := cli.Request(time.Second*60, 2, []byte("send 2"))
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println("req success:", string(reply), err)
	//result: sum duration: [3.1033841s], avg time: [31.033µs], num: [100000], mode: [AsyncRun]
	utils.AsyncRun(100000, func() {
		reply, err := cli.Request(time.Second*60, 2, []byte("send 2"))
		if err != nil {
			fmt.Println(err, string(reply))
			os.Exit(1)
		}
		//fmt.Println(string(reply))
	}).Debug()
}

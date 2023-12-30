package node

import (
	"fmt"
	utils "github.com/Li-giegie/go-utils"
	"log"
	"os"
	"testing"
	"time"
)

func TestClient_Registration(t *testing.T) {
	cli := NewClient(DEFAULT_ServerAddress,
		WithClientId(3790),
		WithClientLocalIpAddr("127.0.0.1:6667"),
		WithClientKeepAlive(time.Second*2))
	authReply, err := cli.Connect([]byte("hello"))
	if err != nil {
		t.Error(err, string(authReply))
		return
	}
	cli.HandleFunc(100, func(ctx *Context) {
		fmt.Println("send:", ctx.String())
	})
	cli.HandleFunc(200, func(ctx *Context) {
		fmt.Println("send 2:", ctx.String())
	})
	if badApi, err := cli.Registration(); err != nil {
		fmt.Println("api: ", badApi)
		t.Error(err)
		return
	}
	if err := cli.Run(); err != nil {
		t.Error(err)
	}
}

func TestClientSend(t *testing.T) {
	cli := NewClient(DEFAULT_ServerAddress, WithClientId(6655), WithClientLocalIpAddr("127.0.0.1:6655"))
	authReply, err := cli.Connect([]byte("hello"))
	if err != nil {
		t.Error(err, string(authReply))
		return
	}
	//result: sum duration: [2.7612166s], avg time: [27.612µs], num: [100000], mode: [AsyncRun]
	utils.AsyncRun(1, func() {
		if err = cli.Send(1, []byte("send 1")); err != nil {
			log.Println(err)
		}
	}).Debug()
	if err = cli.Send(1, []byte("send 1")); err != nil {
		log.Println(err)
	}
	if err = cli.Send(2, []byte("send 1")); err != nil {
		log.Println(err)
	}
	if err = cli.Send(10, []byte("send 1")); err != nil {
		log.Println(err)
	}
	if err = cli.Send(20, []byte("send 1")); err != nil {
		log.Println(err)
	}
	if err = cli.Send(100, []byte("send 1")); err != nil {
		log.Println(err)
	}
	if err = cli.Send(200, []byte("send 1")); err != nil {
		log.Println(err)
	}
}

func TestClientRequest(t *testing.T) {
	cli := NewClient(DEFAULT_ServerAddress)
	authReply, err := cli.Connect([]byte("hello"))
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
	reply, err = cli.Request(time.Second*60, 3, []byte("send 3"))
	if err != nil {
		t.Error(err, string(reply))
	}
	fmt.Println(string(reply))

	//result: sum duration: [3.1033841s], avg time: [31.033µs], num: [100000], mode: [AsyncRun]
	utils.AsyncRun(1, func() {
		reply, err := cli.Request(time.Second*60, 3, []byte("send 2"))
		if err != nil {
			fmt.Println(err, string(reply))
			os.Exit(1)
		}
		//fmt.Println(string(reply))
	}).Debug()
	//fmt.Println(cli.Request(time.Second*3, 1, []byte("send 1")))
	fmt.Println(cli.Request(time.Second*3, 2, []byte("send 2")))
}

package node

import (
	"fmt"
	"testing"
	"time"
)

var addr = "39.101.193.248:8088"

func TestClient_Auth(t *testing.T) {
	cli := NewClient(DEFAULT_ServerAddress,
		WithClientKeepAlive(time.Second*2))
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
		WithClientLocalIpAddr("0.0.0.0:6667"),
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
	for {
		reply, err := cli.Request(time.Second*3, 200, []byte("hello"))
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Printf("reply: %s\n", reply)
		time.Sleep(time.Second)
		continue
	}

}

func TestClientRequest(t *testing.T) {
	cli := NewClient("39.101.193.248:8088", WithClientLocalIpAddr("0.0.0.0:7788"))
	authReply, err := cli.Connect(DEFAULT_ServerID, []byte("hello"))
	if err != nil {
		t.Error("auth err: ", err, string(authReply))
		return
	}
	apis := []uint32{1, 2, 3, 10, 20, 100, 200}
	for _, api := range apis {
		reply, err := cli.Request(time.Second*2, api, []byte("send 2"))
		if err != nil {
			fmt.Println(api, string(reply), err)
			continue
		}
		fmt.Println(api, string(reply), err)
	}
	return
	//reply, err := cli.Request(time.Second*3, 2, []byte("req 2"))
	//if err != nil {
	//	t.Error(err)
	//	return
	//}
	//fmt.Println("req success:", string(reply), err)
	//var dstId uint64 = 1
	//buf, err := json.Marshal(dstId)
	//if err != nil {
	//	t.Error(err)
	//	return
	//}
	//reply, err = cli.Request(time.Second*3, 3, buf)
	//if err != nil {
	//	t.Error(err)
	//	return
	//}
	//fmt.Println("forward success:", string(reply), err)

}

package node

import (
	"context"
	"fmt"
	utils "github.com/Li-giegie/go-utils"
	"log"
	"strings"
	"testing"
	"time"
)

func newClient(localAddr ...string) ClientI {
	if len(localAddr) == 0 {
		localAddr = []string{"127.0.0.1:8919"}
	}
	cli := NewClient(DEFAULT_ServerAddress,
		WithClientAuthentication([]byte(permit)),
		WithClientLocalIpAddr(localAddr[0]),
	)
	_, err := cli.Connect()
	if err != nil {
		log.Fatalln(err)
	}
	return cli
}

// 测试场景一：认证成功后结束
func TestClientAuthScene1(t *testing.T) {
	cli := NewClient(DEFAULT_ServerAddress,
		WithClientAuthentication([]byte(permit)),
		WithClientId(DEFAULT_ClientID),
	)
	authReply, err := cli.Connect()
	defer cli.Close(true)
	if err != nil {
		t.Error("scene 1 err:", string(authReply), err)
		return
	}
}

// 测试场景二：认证用户在线，认证失败
func TestClientAuthScene2(t *testing.T) {
	go func() {
		cli := NewClient(DEFAULT_ServerAddress,
			WithClientAuthentication([]byte(permit)),
			WithClientLocalIpAddr("127.0.0.1:8919"),
		)
		authReply, err := cli.Connect()
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Println(string(authReply))
		time.Sleep(time.Second * 10)
		defer cli.Close(true)

	}()
	time.Sleep(time.Second * 2)
	cli := NewClient(DEFAULT_ServerAddress, WithClientAuthentication([]byte(permit)))
	authReply, err := cli.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	defer cli.Close(true)
	fmt.Println(string(authReply))
}

func TestClient_Send(t *testing.T) {
	cli := NewClient(DEFAULT_ServerAddress,
		WithClientAuthentication([]byte(permit)),
		WithClientLocalIpAddr("127.0.0.1:9000"),
		WithClientId(20),
	)
	_, err := cli.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	defer cli.Close(true)
	err = cli.Send(sendApi, append([]byte{0}, []byte("你好~")...))
	if err != nil {
		t.Error(err)
		return
	}
	//发送到服务端让服务端发起请求到客户端
	//go TestClientForwardServe(t)
	//time.Sleep(time.Second * 2)
	//toSrvReq, _ := jeans.Encode(msgType_Req, forwardClientListenId, forwardClientHandleApi, []byte("你好~"))
	//fmt.Println(cli.Send(sendApi, toSrvReq))
	//time.Sleep(time.Second * 3)
}

// 测试场景一：请求成功
func TestClientScene1(t *testing.T) {
	cli := NewClient(DEFAULT_ServerAddress,
		WithClientAuthentication([]byte(permit)),
		WithClientLocalIpAddr("127.0.0.1:9000"),
		WithClientId(20),
	)
	_, err := cli.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	defer cli.Close(true)
	resp, err := cli.Request(context.Background(), reqApi, []byte("hello"))
	if err != nil {
		t.Error(err, resp)
		return
	}
	fmt.Println(string(resp))
}

// 测试场景二：请求失败，目的处理函数不存在
func TestClientScene2(t *testing.T) {
	cli := newClient()
	defer cli.Close(true)
	resp, err := cli.Request(context.Background(), 100000, []byte("hello"))
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(err, string(resp))
}

// 测试场景三：请求失败，请求超时
func TestClientScene3(t *testing.T) {
	cli := newClient()
	defer cli.Close(true)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	data := []byte{100}
	resp, err := cli.Request(ctx, reqApi, data)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(err, string(resp))

}

var forwardClientListenId uint64 = 64
var forwardClientHandleApi uint32 = 1
var forwardClientHandleperFormanceTestApi uint32 = 2

// 测试客户端转发服务
func TestClientForwardServe(t *testing.T) {
	cli := NewClient(DEFAULT_ServerAddress, WithClientLocalIpAddr("0.0.0.0:7890"), WithClientId(forwardClientListenId))
	cli.HandleFunc(forwardClientHandleApi, func(id uint64, data []byte) (out []byte, err error) {
		fmt.Println(id, string(data))
		if strings.Contains(string(data), "wait") {
			time.Sleep(time.Second * 3)
		}
		return append([]byte("client handle success reply src data: "), data...), err
	})
	cli.HandleFunc(forwardClientHandleperFormanceTestApi, func(id uint64, data []byte) (out []byte, err error) {
		return data, nil
	})
	_, err := cli.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	cli.Run()
}

// 测试正常流程
func TestClientForwardScene1(t *testing.T) {
	cli := newClient()
	//cli.id = "request-node-client"
	defer cli.Close(true)

	resp, err := cli.Forward(context.Background(), forwardClientListenId, forwardClientHandleApi, []byte("hello"))
	if err != nil {
		t.Error(err, resp)
		return
	}
	fmt.Println(string(resp))
}

// 测试客户端服务不存在
func TestClientForwardScene2(t *testing.T) {
	cli := newClient()
	defer cli.Close(true)
	resp, err := cli.Forward(context.Background(), 10000, 1, nil)
	if err != nil {
		fmt.Println(string(resp), err)
		return
	}
	t.Error("err:")
	fmt.Println(string(resp))
}

// 测试客户端服务不回复
func TestClientForwardScene3(t *testing.T) {
	cli := newClient()
	defer cli.Close(true)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	resp, err := cli.Forward(ctx, forwardClientListenId, forwardClientHandleApi, []byte("wait"))
	if err != nil {
		t.Error(err, resp)
		return
	}
	fmt.Println("resp :", string(resp))
	time.Sleep(time.Second * 3)
}

// 测试客户端服务回复超时
func TestClientForwardScene4(t *testing.T) {
	cli := newClient()
	defer cli.Close(true)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resp, err := cli.Forward(ctx, forwardClientListenId, forwardClientHandleApi, []byte("wait"))
	if err != nil {
		t.Error(err, resp)
		//return
	}
	fmt.Println(string(resp))
	time.Sleep(time.Second * 3)
}

// 测试消息保活
func TestClientTick(t *testing.T) {
	cli := NewClient(DEFAULT_ServerAddress, WithClientKeepAlive(time.Second*3))
	_, err := cli.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	defer cli.Close(true)
	select {}
}

func TestServer_RequestClient(t *testing.T) {
	cli := NewClient(DEFAULT_ServerAddress)
	cli.HandleFunc(1, func(id uint64, data []byte) (out []byte, err error) {
		fmt.Println(id, string(data))
		return nil, nil
	})
	_, err := cli.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	defer cli.Close(true)
	cli.Run()
}

// 测试发送消息
func TestClientSend(t *testing.T) {
	cli := NewClient(DEFAULT_ServerAddress)
	_, err := cli.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	err = cli.Send(sendApi, []byte("send print"))
	fmt.Println(err)
	defer cli.Close(true)
}

// 异步请求测试 请求次数：100000 耗时：2.774963s
func TestClientAsyncRequest(t *testing.T) {
	cli := newClient()
	defer cli.Close(true)
	n := 20000
	utils.AsyncRun(n, func() {
		resp, err := cli.Request(context.Background(), 2, []byte("request msg"))
		if err != nil {
			t.Error(err, resp)
			return
		}
	}).Debug()

}

// 异步转发测试 请求次数：100000 耗时：3.7331478s
func TestClientAsyncForward(t *testing.T) {
	go TestClientForwardServe(t)
	time.Sleep(time.Second * 2)
	cli := newClient()
	defer cli.Close(true)
	n := 10
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	utils.AsyncRun(n, func() {
		resp, err := cli.Forward(ctx, forwardClientListenId, forwardClientHandleperFormanceTestApi, []byte("hello"))
		if err != nil {
			t.Error(err, resp)
			return
		}
		fmt.Println(string(resp))
	}).Debug()
}

func TestClient_Registration(t *testing.T) {
	cli := newClient()
	defer cli.Close(true)
	cli.HandleFunc(1, func(id uint64, data []byte) (out []byte, err error) {
		log.Println(id, string(data), "api handle [1] access success")
		return []byte("api handle [1] access success"), err
	})
	cli.HandleFunc(2, func(id uint64, data []byte) (out []byte, err error) {
		log.Println(id, string(data), "api handle [2] access success")
		return []byte("api handle [2] access success"), err
	})
	cli.HandleFunc(3, func(id uint64, data []byte) (out []byte, err error) {
		log.Println(id, string(data), "api handle [3] access success")
		return []byte("api handle [2] access success"), err
	})
	badApiList, err := cli.Registration()
	if err != nil {
		log.Println(err, badApiList)
		return
	}
	cli.Run()
}

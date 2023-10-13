package node

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"testing"
	"time"
)

// 测试场景一：请求成功
func TestClientScene1(t *testing.T) {
	c := NewClient(DEFAULT_ClientID, DEFAULT_ServerAddress)
	conn, err := c.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	var req ReqScene
	resq, err := conn.Request(context.Background(), req.Api(), req.Hello())
	if err != nil {
		t.Error(err, resq)
		return
	}
	fmt.Println(resq.String())
	conn.Close()
}

// 测试场景二：请求失败，目的api不存在
func TestClientScene2(t *testing.T) {
	c := NewClient(DEFAULT_ClientID, DEFAULT_ServerAddress)
	conn, err := c.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	var req ReqScene
	resq, err := conn.Request(context.Background(), 0, req.Hello())
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(err, resq.String())
	conn.Close()
}

// 测试场景三：请求失败，请求超时
func TestClientScene3(t *testing.T) {
	c := NewClient(DEFAULT_ClientID, DEFAULT_ServerAddress)
	conn, err := c.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	//var req ReqScene
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	resq, err := conn.Request(ctx, ReqScene{}.Api(), nil)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(err, resq.String())
	conn.Close()
}

type ForwardScene struct {
}

func (ForwardScene) Name() string {
	return "ForwardScene-serve"
}

func (ForwardScene) Hello() []byte {
	return []byte("hello ForwardScene-serve")
}

func (ForwardScene) Api() uint32 {
	return 1
}

func (ForwardScene) Handler() HandlerFunc {
	return func(ctx *Context) {
		log.Println("ForwardScene Handler:", ctx.String())
		if len(ctx.Data) != 0 {
			_ = ctx.Write([]byte("ForwardScene success"))
		} else {
			time.Sleep(time.Second * 2)
			_ = ctx.Write([]byte("ForwardScene success"))
		}
	}
}

// 测试客户端转发服务
func TestClientForwardServe(t *testing.T) {
	var service ForwardScene
	client := NewClient(service.Name(), DEFAULT_ServerAddress)
	client.AddRouterI(service)
	conn, err := client.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	conn.ListenAndServe()
}

// 测试正常流程
func TestClientForwardScene1(t *testing.T) {
	c := NewClient("forward-client", DEFAULT_ServerAddress)
	conn, err := c.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	var service ForwardScene
	resq, err := conn.RequestForward(context.Background(), service.Name(), service.Api(), service.Hello())
	if err != nil {
		t.Error(err, resq)
		return
	}
	fmt.Println(resq.String())
}

// 测试客户端服务不存在
func TestClientForwardScene2(t *testing.T) {
	c := NewClient("forward-client", DEFAULT_ServerAddress)
	conn, err := c.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	var service ForwardScene
	resq, err := conn.RequestForward(context.Background(), "1forward-client", service.Api(), service.Hello())
	if err != nil {
		t.Error(err, resq)
		return
	}
	fmt.Println(resq.String())
}

// 测试客户端服务不回复
func TestClientForwardScene3(t *testing.T) {
	c := NewClient("forward-client", DEFAULT_ServerAddress)
	conn, err := c.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	var service ForwardScene
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	resq, err := conn.RequestForward(ctx, service.Name(), service.Api(), nil)
	if err != nil {
		t.Error(err, resq)
		return
	}
	fmt.Println(resq.String())
}

// 测试客户端服务回复超时
func TestClientForwardScene4(t *testing.T) {
	c := NewClient("forward-client", DEFAULT_ServerAddress)
	conn, err := c.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	var service ForwardScene
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resq, err := conn.RequestForward(ctx, service.Name(), service.Api(), nil)
	if err != nil {
		t.Error(err, resq)
		//return
	}
	//fmt.Println(resq.String())
	for {
	}
}

// 测试消息保活
func TestClientTick(t *testing.T) {
	c := NewClient(DEFAULT_ClientID+"-forward", DEFAULT_ServerAddress)
	c.KeepAlive = time.Second * 3
	conn, err := c.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()

	select {}
}

// 异步请求测试 请求次数：100000 耗时：3.0871572s
func TestClientAsyncRequest(t *testing.T) {
	c := NewClient(DEFAULT_ClientID, DEFAULT_ServerAddress)
	conn, err := c.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	var req ReqScene
	n := 10
	t1 := time.Now()
	var w sync.WaitGroup
	w.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			resq, err := conn.Request(context.Background(), req.Api(), req.Hello())
			if err != nil {
				t.Error(err, resq)
				return
			}
			//fmt.Println(resq.String())
			w.Done()
		}()
	}
	w.Wait()
	fmt.Printf("异步请求测试 请求次数：%v 耗时：%v\n", n, time.Since(t1))
	conn.Close()
}

// 异步请求测试 请求次数：100000 耗时：5.4646363s
func TestClientAsyncForward(t *testing.T) {
	c := NewClient("forward-client", DEFAULT_ServerAddress)
	conn, err := c.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	var service ForwardScene
	n := 100000
	var w sync.WaitGroup
	w.Add(n)
	t1 := time.Now()
	for i := 0; i < n; i++ {
		go func() {
			resq, err := conn.RequestForward(context.Background(), service.Name(), service.Api(), service.Hello())
			if err != nil {
				t.Error(err, resq)
				return
			}
			w.Done()
		}()
	}
	w.Wait()
	fmt.Printf("异步请求测试 请求次数：%v 耗时：%v\n", n, time.Since(t1))
	conn.Close()
}

// 测试服务端最大连接数功能
func TestServerMaxConnectNum(t *testing.T) {

	var clients = make([]*Client, 0)

	for i := 0; i < 10000; i++ {
		id := strconv.Itoa(i)
		c1 := NewClient(id, DEFAULT_ServerAddress)
		_, err := c1.Connect(nil)
		if err != nil {
			t.Error(err)
			continue
		}
		//res, err := conn.Request(context.Background(), 1, []byte("i'm "+id))
		//if err != nil {
		//	t.Error(err)
		//	return
		//}
		//fmt.Println(res.String())
		clients = append(clients, c1)
		//time.Sleep(time.Second * 1)
	}

	time.Sleep(time.Second * 5)
	for _, client := range clients {
		client.conn.Close()
	}

}

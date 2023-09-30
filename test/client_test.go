package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"sync"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	c := node.NewClient(node.DEFAULT_ClientID, node.DEFAULT_ServerAddress)
	conn, err := c.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	w := sync.WaitGroup{}
	w.Add(50000)
	t1 := time.Now()
	for i := 0; i < 50; i++ {
		go func() {
			resq, err := conn.Request(context.Background(), 2, []byte("hello node!"))
			if err != nil {
				t.Error(err, resq)
				w.Done()
				return
			}
			fmt.Println(resq.String())
			w.Done()
		}()
	}
	w.Wait()
	fmt.Println(time.Since(t1))
	//fmt.Println(resq.String())
	conn.Close()
}

func TestClientHandler(t *testing.T) {
	c := node.NewClient(node.DEFAULT_ClientID, node.DEFAULT_ServerAddress)
	conn, err := c.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}

	conn.RouteManager.AddRoute(2, func(ctx *node.Context) {
		fmt.Println(ctx.String())
		ctx.Write([]byte("forward success"))
	})

	defer conn.Close()

	conn.ListenAndServe()
}
func TestClientForward(t *testing.T) {
	c := node.NewClient(node.DEFAULT_ClientID+"-forward", node.DEFAULT_ServerAddress)
	conn, err := c.Connect(nil)
	if err != nil {
		t.Error(err)
		return
	}
	var w sync.WaitGroup
	w.Add(5)
	t1 := time.Now()
	for i := 0; i < 5; i++ {
		go func() {
			resq, err := conn.RequestForward(context.Background(), node.DEFAULT_ClientID, 2, []byte("hello node!"))
			if err != nil {
				t.Error(err, resq)
				w.Done()
				return
			}
			w.Done()
			fmt.Println(resq.String())
		}()

	}
	w.Wait()
	fmt.Println(time.Since(t1))
	conn.Close()
}

func TestClientTick(t *testing.T) {
	c := node.NewClient(node.DEFAULT_ClientID+"-forward", node.DEFAULT_ServerAddress)
	c.KeepAlive = time.Second * 3
	c.AddRoute(1, func(ctx *node.Context) {
		ctx.Write([]byte("ok"))
	})
	conn, err := c.Connect(nil)

	if err != nil {
		t.Error(err)
		return
	}

	defer conn.Close()

	select {}
}

package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/panjf2000/ants/v2"
	"log"
	"sync"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	c, err := node.DialTCP(1, "127.0.0.1:8080", 2)
	if err != nil {
		t.Error(err)
		return
	}

	conn, err := c.AuthenticationWithServer(context.Background(), []byte{})
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()
	if err = conn.Tick(time.Second, time.Second*3, time.Second*6, true); err != nil {
		t.Error(err)
		return
	}

	go func() {
		time.Sleep(time.Second)
		wg := sync.WaitGroup{}
		p, _ := ants.NewPool(10000)
		t1 := time.Now()
		for i := 0; i < 100000; i++ {
			wg.Add(1)
			err = p.Submit(func() {
				defer wg.Done()
				resp, err := conn.Request(context.Background(), 1, nil)
				if err != nil {
					t.Error(err, resp, len(resp))

					return
				}
				//fmt.Println(resp)
			})
			if err != nil {
				t.Error(err)
			}
		}
		wg.Wait()
		fmt.Println(time.Since(t1))
		//conn.Close()
	}()

	if err = conn.Serve(); err != nil {
		t.Error(err)
		log.Println("serve err")
		return
	}
}

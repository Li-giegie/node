package main

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/example"
	"log"
	"sync"
	"time"
)

func main() {
	serverNode()
}

func serverNode() {
	srv := node.NewServer(example.SERVER_ADDR)

	//先发送往服务节点，服务节点转发到client_node2，不推荐此做法发，应该用注册或者直接转发功能实现
	srv.HandleFunc(example.SERVER_FORWARD_API, func(ctx *node.Context) {
		// if ... {
		// ctx.ReplyErr(errors.New("..."),[]byte("..."))
		//}
		// todo: ctx.Data() ......

		conn1, ok := srv.GetConnect(example.CLIENT1_ID)
		if !ok {
			_ = ctx.ReplyErr(node.ErrConnNotExist, nil)
			return
		}
		conn2, ok := srv.GetConnect(example.CLIENT2_ID)
		if !ok {
			_ = ctx.ReplyErr(node.ErrConnNotExist, nil)
			return
		}
		type result struct {
			reply []byte
			err   error
			id    uint64
		}
		var replyChan = make(chan *result, 2)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			reply, err := conn1.Request(time.Second*3, example.CLIENT1_2_GROUPAPI, ctx.Data())
			replyChan <- &result{reply, err, example.CLIENT1_ID}
		}()
		go func() {
			defer wg.Done()
			reply, err := conn2.Request(time.Second*3, example.CLIENT1_2_GROUPAPI, ctx.Data())
			replyChan <- &result{reply, err, example.CLIENT2_ID}
		}()
		wg.Wait()
		c1 := <-replyChan
		c2 := <-replyChan
		close(replyChan)
		if c1.err != nil && c2.err != nil {
			_ = ctx.ReplyErr(fmt.Errorf("client_node1 err: %v\nclient_node2 err: %v", c1.err, c2.err), nil)
			return
		} else if c1.err != nil {
			_ = ctx.ReplyErr(fmt.Errorf("client_node1 err: %v", c1.err), nil)
			return
		} else if c2.err != nil {
			_ = ctx.ReplyErr(fmt.Errorf("client_node2 err: %v", c2.err), nil)
			return
		}
		_ = ctx.Reply([]byte("forward to client_node1、client_node2 success"))
	})
	//启动服务 可选参数debug：true 打印输出
	if err := srv.ListenAndServer(true); err != nil {
		log.Println(err)
	}
}

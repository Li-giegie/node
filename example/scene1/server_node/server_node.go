package main

import (
	"errors"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/example"
	"log"
)

func main() {
	serverNode()
}

func serverNode() {
	srv := node.NewServer(example.SERVER_ADDR,
		// 设置节点ID
		node.WithSrvId(example.SERVER_ID),
		// 最大连接数量
		node.WithSrvMaxConnectNum(10000),
		// 连接时间相关
		node.WithSrvTimeParameters(node.ServerTimeParameters{
			//最大连接空闲时间
			MaxConnectionIdle: node.DEFAULT_ConnectionIdle,
			//检查连接是否有效间隔时间
			CheckInterval: node.DEFAULT_CheckInterval,
		}),
		node.WithSrvGoroutineParameters(node.ServerGoroutineParameters{
			//开启的Goroutine数量
			GoroutineNum: node.DEFAULT_MIN_GOROUTINE,
			//扩容Goroutine最大数量，MaxGoroutine > GoroutineNum 有效
			MaxGoroutine: node.DEFAULT_MAX_GOROUTINE,
		}),
		// 入参数：id 发起者的id，data 发起者携带的数据 ，返回参数：reply 回复的内容，err 为nil表示认证通过。否则认证失败
		node.WithSrvAuthentication(func(id uint64, data []byte) (reply []byte, err error) {
			if string(data) == "permit" {
				return nil, nil
			}
			return []byte("invalid argument"), errors.New("deny")
		}),
	)
	//用于仅接受处理函数
	srv.HandleFunc(example.SERVER_SEND_API, func(ctx *node.Context) {
		fmt.Println("1000 handle: ", string(ctx.Data()))
	})
	//用于回复处理函数
	srv.HandleFunc(example.SERVER_REQUEST_API, func(ctx *node.Context) {
		// if ... {
		// ctx.ReplyErr(errors.New("..."),[]byte("..."))
		//}
		fmt.Println("1001 handle: ", string(ctx.Data()))
		_ = ctx.Reply(append([]byte("roger that:"), ctx.Data()...))
	})
	//启动服务 可选参数debug：true 打印输出
	if err := srv.ListenAndServer(true); err != nil {
		log.Println(err)
	}
}

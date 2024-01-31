package main

import (
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/example"
	"log"
)

func main() {
	serverNode()
}

func serverNode() {
	srv := node.NewServer(example.SERVER_ADDR, node.WithSrvId(example.SERVER_ID))

	//先发送往服务节点，服务节点转发到client_node2，不推荐此做法发，应该用注册或者直接转发功能实现
	srv.HandleFunc(example.SERVER_FORWARD_API, func(ctx *node.Context) {
		// if ... {
		// ctx.ReplyErr(errors.New("..."),[]byte("..."))
		//}
		// todo: ctx.Data() ......

		conn, ok := srv.GetConnect(example.CLIENT1_ID)
		if !ok {
			_ = ctx.ReplyErr(node.ErrConnNotExist, nil)
			return
		}
		//req: c2 --> s --> s handle --> c1; res:c1 --> s --> s handle --> c2
		//当前节点发起转发，回复直接回复给请求节点，当tcp链路不通会返回到当前节点错误，也可以用request方法（不推荐）
		//提供重写api，数据能力
		if err := conn.Forward(ctx, ctx.Api(), ctx.Data()); err != nil {
			_ = ctx.ReplyErr(node.ErrConnNotExist, nil)
			return
		}
	})
	//启动服务 可选参数debug：true 打印输出
	if err := srv.ListenAndServer(true); err != nil {
		log.Println(err)
	}
}

package main

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"log"
	"net"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "0.0.0.0:8000")
	if err != nil {
		log.Fatalln(err)
	}
	c := node.NewClient(conn, node.CliConf{
		ReaderBufSize:   4096,
		WriterBufSize:   4096,
		WriterQueueSize: 1024,
		MaxMsgLen:       0xffffff,
		ClientIdentity: &node.ClientIdentity{
			Id:            8001,
			RemoteAuthKey: []byte("hello"),
			Timeout:       time.Second * 3,
		},
	})
	exitC := make(chan struct{}, 1)
	// 通过认证后连接正式建立的回调,同步调用
	c.AddOnConnection(func(conn iface.Conn) {
		log.Println("OnConnection id", conn.RemoteId(), "type", conn.NodeType())
	})
	// 收到内置标准类型消息的回调,同步调用
	c.AddOnMessage(func(ctx iface.Context) {
		log.Println("OnMessage", string(ctx.Data()))
		rdata := fmt.Sprintf("from %d data %s", c.Id(), ctx.Data())
		// 回复消息
		ctx.Reply([]byte(rdata))
		// 回复错误
		//ctx.ErrReply(nil,errors.New("invalid request"))
	})
	// 收到自定义消息类型的回调,同步调用
	c.AddOnCustomMessage(func(ctx iface.Context) {
		log.Println("OnCustomMessage", string(ctx.Data()))
	})
	// 连接关闭的回调,同步调用, 该回调通常用于协议
	c.AddOnClosed(func(conn iface.Conn, err error) {
		log.Println("OnClosed", conn.IsClosed(), err)
		exitC <- struct{}{}
	})
	// 收到非本地节点的消息并且没有路由时触发，同步调用, 服务端节点如果该回调为空，则默认回复节点不存在错误 客户端节点不应该收到目的节点非本地节点的消息，该回调为空，没有默认行为，丢弃该消息
	c.AddOnNoRouteMessage(func(ctx iface.Context) {
		log.Println("OnNoIdMessage", ctx.String())
	})
	if err = c.Start(); err != nil {
		log.Println(err)
		return
	}
	resp, err := c.Request(context.Background(), []byte("ping"))
	if err != nil {
		println(err)
	} else {
		println(string(resp))
	}
	c.Close()
	<-exitC
}

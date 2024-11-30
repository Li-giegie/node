package main

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"log"
	"net"
	"time"
)

func main() {
	// 创建服务端
	s := node.NewServer(&node.Identity{Id: 8000, Key: []byte("hello"), Timeout: time.Second * 6}, nil)
	// 通过认证后连接正式建立的回调,同步调用
	s.AddOnConnect(func(conn iface.Conn) {
		log.Println("OnConnection id", conn.RemoteId())
	})
	// 收到内置标准类型消息的回调,同步调用
	s.AddOnMessage(func(ctx iface.Context) {
		log.Println("OnMessage", string(ctx.Data()))
		rdata := fmt.Sprintf("from %d data %s", s.Id(), ctx.Data())
		// 回复消息
		ctx.Reply([]byte(rdata))
		// 回复错误
		//ctx.ErrReply(nil,errors.New("invalid request"))
	})
	// 收到自定义协议消息类型的回调，协议适用于扩展功能，并不是用来区别场景的，所有场景都应该在AddOnMessage回调中实现,同步调用
	//s.AddOnProtocolMessage(255,func(ctx iface.Context) {
	//	log.Println("OnCustomMessage", string(ctx.Data()))
	//})
	// 连接关闭的后回调,同步调用, 该回调通常用于协议
	s.AddOnClose(func(conn iface.Conn, err error) {
		log.Println("OnClosed", conn.RemoteId(), err)
	})
	l, err := net.Listen("tcp", "0.0.0.0:8000")
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("start success")
	if err = s.Serve(l); err != nil {
		log.Println(err)
	}
}

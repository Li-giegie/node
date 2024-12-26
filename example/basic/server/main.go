package main

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/message"
	"log"
	"net"
	"time"
)

func main() {
	// 创建服务端
	s := node.NewServer(&node.Identity{Id: 8000, Key: []byte("hello"), AuthTimeout: time.Second * 6})
	// accept 一个连接时触发回调，allow 返回值为false时断开连接
	s.OnAccept(func(conn net.Conn) (allow bool) {
		log.Println("OnAccept", conn.RemoteAddr().String())
		return true
	})
	// 通过认证后连接正式建立的回调,同步调用
	s.OnConnect(func(conn iface.Conn) {
		log.Println("OnConnection conn id:", conn.RemoteId())
	})
	// 收到消息的回调,同步调用
	s.OnMessage(func(ctx iface.Context) {
		log.Println("OnMessage", string(ctx.Data()))
		if ctx.Type() != message.MsgType_Default {
			ctx.Response(message.StateCode_MessageTypeInvalid, nil)
			return
		}
		rdata := fmt.Sprintf("from %d data %s", s.Id(), ctx.Data())
		// 回复消息
		ctx.Response(200, []byte(rdata))
	})
	// 连接断开时回调
	s.OnClose(func(conn iface.Conn, err error) {
		log.Println("OnClosed", conn.RemoteId(), err)
	})
	// 侦听并启动
	err := s.ListenAndServe("0.0.0.0:8000")
	if err != nil {
		log.Println(err)
	}
}

package main

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/ctx"
	"github.com/Li-giegie/node/pkg/message"
	"log"
	"net"
)

func main() {
	// 创建一个节点Id为8000的节点
	s := node.NewServerOption(8000)
	// accept 一个连接时触发回调，allow 返回值为false时断开连接
	s.OnAccept(func(conn net.Conn) (allow bool) {
		log.Println("OnAccept", conn.RemoteAddr().String())
		return true
	})
	// 通过认证后连接正式建立的回调,同步调用
	s.OnConnect(func(conn conn.Conn) {
		log.Println("OnConnection conn id:", conn.RemoteId())
	})
	// 收到消息的回调,同步调用
	s.OnMessage(func(ctx ctx.Context) {
		log.Println("OnMessage", string(ctx.Data()))
		if ctx.Type() != message.MsgType_Default {
			ctx.Response(message.StateCode_MessageTypeInvalid, nil)
			return
		}
		rdata := fmt.Sprintf("from %d data %s", s.NodeId(), ctx.Data())
		// 回复消息
		ctx.Response(200, []byte(rdata))
	})
	// 连接断开时回调
	s.OnClose(func(conn conn.Conn, err error) {
		log.Println("OnClosed", conn.RemoteId(), err)
	})
	// 侦听并启动
	err := s.ListenAndServe("0.0.0.0:8000")
	if err != nil {
		log.Println(err)
	}
}

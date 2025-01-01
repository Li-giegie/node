package main

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
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
	s.OnConnect(func(conn conn.Conn) (next bool) {
		return true
	})
	// 所有类型的消息都会进入,同步调用
	s.OnMessage(func(r responsewriter.ResponseWriter, m *message.Message) (next bool) {
		r.Response(message.StateCode_Success, []byte(fmt.Sprintf("response from %d: ok", s.NodeId())))
		return true
	})
	s.OnClose(func(conn conn.Conn, err error) (next bool) {
		return true
	})
	// 侦听并启动
	err := s.ListenAndServe("0.0.0.0:8000")
	if err != nil {
		log.Println(err)
	}
}

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
	l, err := net.Listen("tcp", "0.0.0.0:8000")
	if err != nil {
		log.Fatalln(err)
	}
	// 创建服务端
	s := node.NewServer(l, node.SrvConf{
		// 节点的id、秘钥、认证时间，超过这个时长没有认真成功则断开连接
		Identity: &node.Identity{
			Id:          8000,
			AuthKey:     []byte("hello"),
			AuthTimeout: time.Second * 3,
		},
		// 一次发送的最大消息长度字节，这里为3个字节
		MaxMsgLen: 0xffffff,
		// 发送消息的队列大小，推荐值10 ~ 1024
		WriterQueueSize: 1024,
		// 连接读缓冲区大小
		ReaderBufSize: 4096,
		// 连接写缓冲区大小
		WriterBufSize: 4096,
		// 服务端允许最大的连接数
		MaxConns: 128,
		// 服务端建立连接超过最大连接数时，进入休眠，休眠时长递增增长，最大休眠时长
		MaxListenSleepTime: time.Second * 30,
		// 每次休眠的递增的步长，直到最大值后不在递增
		ListenStepTime: time.Second,
	})
	// 通过认证后连接正式建立的回调,同步调用
	s.AddOnConnection(func(conn iface.Conn) {
		log.Println("OnConnection id", conn.RemoteId(), "type", conn.NodeType())
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
	// 收到自定义消息类型的回调,同步调用
	s.AddOnCustomMessage(func(ctx iface.Context) {
		log.Println("OnCustomMessage", string(ctx.Data()))
	})
	// 连接关闭的后回调,同步调用, 该回调通常用于协议
	s.AddOnClosed(func(conn iface.Conn, err error) {
		log.Println("OnClosed", conn.IsClosed(), err)
	})
	// AddOnNoRouteMessage 收到非本地节点的消息并且没有路由时触发，同步调用, 当节点是服务端节点时，如果该回调为空，则默认回复节点不存在错误 当节点是客户端节点时，不应该收到目的节点非本地节点的消息，该回调为空时也没有默认行为，丢弃该消息

	if err = s.Serve(); err != nil {
		log.Println(err)
	}
}

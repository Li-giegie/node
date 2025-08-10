# node

## 介绍
node是一个Go（Golang）编写的轻量级TCP框架，node帮助您轻松、快速构建TCP服务器。

特征：
- 支持请求响应模型
- 支持多服务端节点桥接组网内部实现动态路由协议
- 并发100w/s 请求响应

## 传输协议
```go
type Message struct {
Type   uint8  //消息类型，不同的协议该值不同
Hop    uint8  //消息的跳数
Id     uint32 //消息唯一标识
SrcId  uint32 //源节点
DestId uint32 //目的节点
Data   []byte //消息内容
}
```
<table >
  <tr>
    <th rowspan="2" >Header 20Byte</th>
    <td >Type 1Byte</td>
    <td >Hop 1Byte</td>
    <td >Id 4Byte</td>
    <td >SrcId 4Byte</td>
    <td >DestId 4Byte</td>
    <td >DataLength 4Byte</td>
  </tr>
  <tr >
    <td align="center" colspan="6">CheckSum 2Byte</td>
  </tr>
  <tr >
    <td align="center" colspan="7">Data</td>
  </tr>
</table>

## 安装
```
go get -u github.com/Li-giegie/node@latest
```
## 快速开始
### Server
[Server 完整的示例](example/base/server/main.go)
```go
package main

import (
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/reply"
	"github.com/Li-giegie/node/pkg/server"
	"log"
	"net"
)

func main() {
	srv := node.NewServerOption(1)
	server.OnAccept(func(conn net.Conn) (next bool) {
		log.Println("OnAccept", conn.RemoteAddr())
		return true
	})
	server.OnConnect(func(conn *conn.Conn) (next bool) {
		log.Println("OnConnect", conn.RemoteAddr())
		return true
	})
	server.OnMessage(func(r *reply.Reply, m *message.Message) (next bool) {
		log.Println("OnMessage", m.String())
		r.String(message.StateCode_Success, "pong")
		return true
	})
	server.OnClose(func(conn *conn.Conn, err error) (next bool) {
		log.Println("OnClose", conn.RemoteAddr())
		return true
	})
	log.Println("listen: 7890")
	err := srv.ListenAndServe(":7890", nil)
	if err != nil {
		log.Fatal(err)
	}
}
```

### Client
[Client 完整的示例](example/base/client/main.go)
```go
package main

import (
	"context"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/client"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/reply"
	"log"
)

func main() {
	c := node.NewClientOption(2, 1)
	client.OnMessage(func(r *reply.Reply, m *message.Message) (next bool) {
		log.Println(m.String())
		r.Write(message.StateCode_Success, m.Data)
		return true
	})
	client.OnClose(func(conn *conn.Conn, err error) (next bool) {
		log.Println("OnClose", err)
		return true
	})
	err := c.Connect("tcp://127.0.0.1:7890", nil)
	if err != nil {
		log.Fatal("connect err:", err)
		return
	}
	log.Println("connect: 7890")
	defer c.Close()
	if err = c.Send([]byte("hello")); err != nil {
		log.Println("send err:", err)
		return
	}
	code, data, err := c.Request(context.TODO(), []byte("world"))
	if err != nil {
		log.Println("request err:", err)
		return
	}
	log.Println("code:", code, "data:", string(data))
}
```

## 基准
测试环境：联想小新 Pro16 锐龙版

测试包：github.com/Li-giegie/node/test

文件：
- bench_echo_server_test.go
- bench_echo_client_test.go

测试函数：
- TestServer
- BenchmarkEchoRequest 同步请求
- BenchmarkEchoRequestGo 并发请求
```go
go test -run none -bench BenchmarkEchoRequest -benchmem
goos: windows
goarch: amd64
pkg: github.com/Li-giegie/node/test
cpu: AMD Ryzen 5 5600H with Radeon Graphics
BenchmarkEchoRequest-12            18549             65039 ns/op             186 B/op          6 allocs/op
BenchmarkEchoRequestGo-12        1000000              1619 ns/op             393 B/op          7 allocs/op
```

## 功能
```
单服务端节点
+---------------------------+
|        Server node        |
|           /    \          |
|          /      \         |  
|     Client      Client    |
+---------------------------+
```
```
多服务端节点互联

                 +---------------------------+
                 |        ServerNode         |
                 |           /    \          |
                 |          /      \         |  
                 |       Node      Node      |
                 +---------------------------+
                /                             \
               /                               \
+-------------/-------------+       +-----------\---------------+
|         ServerNode        |       |         ServerNode        |
|           /    \          |       |           /    \          |
|          /      \         |       |          /      \         |  
|       Node       Node     |       |       Node       Node     |
+---------------------------+       +---------------------------+
```

## 协议
[关于协议的进一步使用 README](docs/protocol)

## 作者邮箱
[859768560@qq.com](https://mail.qq.com/cgi-bin/loginpage?s=logout)


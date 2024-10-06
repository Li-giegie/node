# node

## 介绍
node是一个Go（Golang）编写的轻量级TCP框架，node帮助您轻松、快速构建TCP服务器。

特征：
- [Message](#传输协议)协议实现通信
- 支持请求响应模型
- 支持多服务端节点桥接组网
- 并发100w/s 请求响应

## 传输协议
```go
type Message struct {
  Type   uint8
  Id     uint32
  SrcId  uint32
  DestId uint32
  Data   []byte
}
```
<table >
  <tr>
    <th rowspan="2" >Header 13Byte</th>
    <td >Typ 1Byte</td>
    <td >Id 4Byte</td>
    <td >SrcId 4Byte</td>
    <td >DestId 4Byte</td>
    <td >DataLength 4Byte</td>
  </tr>
  <tr >
    <td align="center" colspan="5">CheckSum 2Byte</td>
  </tr>
  <tr >
    <td align="center" colspan="6">Data</td>
  </tr>
</table>

## 安装
```
go get -u github.com/Li-giegie/node@latest
```
## 快速开始
Handler接口负责连接的生命周期，下文对Handler接口进行了介绍，默认接口生命周期是同步调用。
```go
type Handler interface {
    Connection(conn common.Conn)
    Handle(ctx common.Context)
    ErrHandle(msg *common.Message, err error)
    CustomHandle(ctx common.Context)
    Disconnect(id uint32, err error)
}
```
1. Connection 连接第一次建立成功回调
2. Handle 接到标准类型消息会被触发，如果该回调阻塞将阻塞当前节点整个生命周期回调（在同步调用模式中如果在这个回调中发起请求需要另外开启协程否则会陷入阻塞，无法接收到消息），框架并没有集成协程池，第三方框架众多，合理选择
3. ErrHandle 当收到：超过限制长度的消息0xffffff、校验和错误、超时、服务节点返回的消息但没有接收的消息都会在这里触发回调
4. CustomHandle 自定义消息类型处理，框架内部默认集成了多种消息类型，当需要一些特定的功能时可以自定义消息类型，例如心跳消息，只需把消息类型声明成框架内部不存在的类型，框架看到不认识的消息就会回调当前函数
5. Disconnect 连接断开会被触发
### Server
```go
package test

import (
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"log"
	"testing"
	"time"
)

type Handler struct {
	*node.Server
}

func (h Handler) Connection(conn common.Conn) {
	log.Println("connection", conn.RemoteId())
}

func (h Handler) Handle(ctx common.Context) {
	ctx.Reply([]byte("pong"))
}

func (h Handler) ErrHandle(ctx common.ErrContext, err error) {
	log.Println("ErrHandle", err, ctx.String())
}

func (h Handler) CustomHandle(ctx common.CustomContext) {
	log.Println("CustomHandle", ctx.String())
}

func (h Handler) Disconnect(id uint32, err error) {
	log.Println("Disconnect", id, err)
}

func TestServer(t *testing.T) {
	srv, err := node.ListenTCP("0.0.0.0:8000", &node.Identity{
		Id:            8000,
		AccessKey:     []byte("hello"),
		AccessTimeout: time.Second * 6,
	})
	if err != nil {
		t.Error(err)
		return
	}
	defer srv.Close()
	if err = srv.Serve(&Handler{srv}); err != nil {
		t.Error(err)
		return
	}
}
```

### Client
```go
package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"log"
	"testing"
	"time"
)

type CliHandler struct {}

func (h CliHandler) Connection(conn common.Conn) {
	log.Println("Handle", conn.RemoteId())
}

func (h CliHandler) Handle(ctx common.Context) {
	log.Println("Handle", ctx.String())
}

func (h CliHandler) ErrHandle(ctx common.ErrContext, err error) {
	log.Println("ErrHandle", err, ctx.String())
}

func (h CliHandler) CustomHandle(ctx common.CustomContext) {
	log.Println("CustomHandle", ctx.String())
}

func (h CliHandler) Disconnect(id uint32, err error) {
	fmt.Println("Disconnect", id, err)
}

func TestClient(t *testing.T) {
	conn, err := node.DialTCP(
		"0.0.0.0:8000",
		&node.Identity{
			// Local node Id
			Id:            8001,
			// Remote Node Access key
			AccessKey:     []byte("hello"),
			// Timeout
			AccessTimeout: time.Second * 6,
		},
		&CliHandler{},
	)
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	res, err := conn.Request(ctx, []byte("ping"))
	if err != nil {
		t.Error(err)
		return
	}
	println(string(res))
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

![单域](./.README_images/single.png)

多域间节点互相通信如下

![多域](./.README_images/multiple.png)

[example示例](example)

## 协议
[关于协议的进一步使用 README](protocol/README.md)

## 作者邮箱
[859768560@qq.com](https://mail.qq.com/cgi-bin/loginpage?s=logout)

## 更新迭代
* 增加功能

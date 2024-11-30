# node

## 介绍
node是一个Go（Golang）编写的轻量级TCP框架，node帮助您轻松、快速构建TCP服务器。

特征：
- 支持请求响应模型
- 支持多服务端节点桥接组网
- 节点去中心化
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
[Server 完整的示例](example/basic/server/main.go)
```go
func TestServer(t *testing.T) {
	srv := node.NewServer(&node.Identity{Id: 8000, Key: []byte("hello"), Timeout: time.Second * 6}, nil)
	srv.AddOnConnect(func(conn iface.Conn) {
		log.Println("OnConnection", conn.RemoteId())
	})
	srv.AddOnMessage(func(ctx iface.Context) {
		log.Println("OnMessage", ctx.String())
		fmt.Println(ctx.Reply(ctx.Data()))
	})
	l, err := net.Listen("tcp", "0.0.0.0:8000")
	if err != nil {
		t.Error(err)
		return
	}
	if err = srv.Serve(l); err != nil {
		t.Error(err)
	}
}
```

### Client
[Client 完整的示例](example/basic/client/main.go)
```go
func TestClient(t *testing.T) {
	netConn, err := net.Dial("tcp", "0.0.0.0:8000")
	if err != nil {
		t.Error(err)
		return
	}
	stopC := make(chan struct{})
	c := node.NewClient(8001, &node.Identity{Id: 8000, Key: []byte("hello"), Timeout: time.Second * 6}, nil)
	c.AddOnMessage(func(ctx iface.Context) {
		fmt.Println(ctx.String())
		ctx.Reply(ctx.Data())
	})
	c.AddOnClosed(func(conn iface.Conn, err error) {
		stopC <- struct{}{}
	})
	conn, err := c.Start(netConn)
	if err != nil {
		t.Error(err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	res, err := conn.Request(ctx, []byte("ping"))
	fmt.Println(string(res), err)
	_ = conn.Close()
	<-stopC
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
[关于协议的进一步使用 README](protocol/README.md)

## 作者邮箱
[859768560@qq.com](https://mail.qq.com/cgi-bin/loginpage?s=logout)


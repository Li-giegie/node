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
### Server

```go
func TestServer(t *testing.T) {
	l, err := net.Listen("tcp", "0.0.0.0:8000")
	if err != nil {
		t.Error(err)
		return
	}
	srv := node.NewServer(l, &node.SrvConf{
		// 节点的身份信息
		Identity: &node.Identity{
			// 唯一ID
			Id:          1,
			// 节点的秘钥，想与该节点建立连接需要提供相同的秘钥
			AuthKey:     []byte("hello"),
			// 认证过程超时
			AuthTimeout: time.Second * 6,
		},
		// 最大消息长度
		MaxMsgLen:          0xffffff,
		// 写入队列大小 Request、Write....发送时进入队列
		WriterQueueSize:    1024,
		// 读缓冲区大小
		ReaderBufSize:      4096,
		// 写缓冲区大小
		WriterBufSize:      4096,
		// 最大连接数 > 0有效 
		MaxConns:           0,
		// 超过最大连接数时，进入休眠的最大时间，按照步长递增
		MaxListenSleepTime: time.Minute,
		// 超过限制连接数量，递增休眠步长，直到达到最大休眠时长后停止递增
		ListenStepTime:     time.Second,
	})
	// 以下函数均为同步调用
	// 建立连接回调
	srv.OnConnection = func(conn node.Conn) {
		log.Println("OnConnection", conn.RemoteId())
	}
	// 收到消息回调
	srv.OnMessage = func(ctx node.Context) {
		log.Println("OnMessage", ctx.String())
		ctx.Reply(ctx.Data())
	}
	// 收到自定义消息回调
	srv.OnCustomMessage = func(ctx node.CustomContext) {
		log.Println("OnCustomMessage", ctx.String())
	}
	// 断开连接回调
	srv.OnClose = func(id uint32, err error) {
		log.Println("OnClose", id, err)
	}
	defer srv.Close()
	// 阻塞启动服务
	if err = srv.Serve(); err != nil {
		t.Error(err)
		return
	}
}

```

### Client

```go
func TestClient(t *testing.T) {
	conn, err := net.Dial("tcp", "0.0.0.0:8000")
	if err != nil {
		t.Error(err)
		return
	}
	// 关闭管道
	stopC := make(chan struct{})
	c := node.NewClient(conn, &node.CliConf{
		ReaderBufSize:   4096,
		WriterBufSize:   4096,
		WriterQueueSize: 1024,
		MaxMsgLen:       0xffffff,
		ClientIdentity: &node.ClientIdentity{
			Id:            1234,
			RemoteAuthKey: []byte("hello"),
			Timeout:       time.Second * 6,
		},
	})
	c.OnConnection = func(conn node.Conn) {
		log.Println("OnConnection", conn.RemoteId())
	}
	c.OnMessage = func(ctx node.Context) {
		log.Println("OnMessage", ctx.String())
	}
	c.OnClose = func(id uint32, err error) {
		log.Println("OnClose", id, err)
		stopC <- struct{}{}
	}
	if err = c.Start(); err != nil {
		log.Fatalln(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	// 发起一个请求
	res, err := c.Request(ctx, []byte("ping"))
	if err != nil {
		t.Error(err)
		return
	}
	println(string(res))
	c.Close()
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

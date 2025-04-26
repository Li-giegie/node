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
func main() {
	// 创建8000服务端节点
	s := node.NewServerOption(8000)
	// OnAccept 注册全局OnAccept回调函数，net.Listen.Accept之后第一个回调函数，同步调用
	s.OnAccept(func(conn net.Conn) (next bool) {
		log.Println("OnAccept", conn.RemoteAddr().String())
		return true
	})
	// OnConnect 注册全局OnConnect回调函数，OnAccept之后的回调函数，同步调用
	s.OnConnect(func(conn conn.Conn) (next bool) {
		log.Println("OnConnect", conn.RemoteAddr().String())
		return true
	})
	// OnMessage 注册全局OnMessage回调函数，OnConnect之后每次收到请求时的回调函数，同步调用
	s.OnMessage(func(r responsewriter.ResponseWriter, m *message.Message) (next bool) {
		log.Println("OnMessage", m.String())
		return true
	})
	// OnClose 注册OnClose回调函数，连接被关闭后的回调函数
	s.OnClose(func(conn conn.Conn, err error) (next bool) {
		log.Println("OnClose", conn.RemoteAddr().String())
		return true
	})
	// Register 注册实现了handler.Handler的处理接口，该接口的回调函数在OnAccept、OnConnect、OnMessage、OnClose之后被回调
	s.Register(message.MsgType_Default, &handler.Default{
		OnAcceptFunc:  nil,
		OnConnectFunc: nil,
		OnMessageFunc: func(r responsewriter.ResponseWriter, m *message.Message) {
			log.Println("OnAcceptFunc")
			r.Response(message.StateCode_Success, []byte(fmt.Sprintf("response from %d: ok", s.NodeId())))
		},
		OnCloseFunc: nil,
	})
	// 侦听并开启服务
	err := s.ListenAndServe("0.0.0.0:8000")
	if err != nil {
		log.Println(err)
	}
}
```

### Client
[Client 完整的示例](example/base/client/main.go)
```go
func main() {
	// 创建一个节点为8081的节点
	c := node.NewClientOption(8081, 8000)
	// OnAccept 注册全局OnAccept回调函数，net.Listen.Accept之后第一个回调函数，同步调用
	c.OnAccept(func(conn net.Conn) (next bool) {
		log.Println("OnAccept", conn.RemoteAddr().String())
		return true
	})
	// OnConnect 注册全局OnConnect回调函数，OnAccept之后的回调函数，同步调用
	c.OnConnect(func(conn conn.Conn) (next bool) {
		log.Println("OnConnect", conn.RemoteAddr().String())
		return true
	})
	// OnMessage 注册全局OnMessage回调函数，OnConnect之后每次收到请求时的回调函数，同步调用
	c.OnMessage(func(r responsewriter.ResponseWriter, m *message.Message) (next bool) {
		log.Println("OnMessage", m.String())
		return true
	})
	// OnClose 注册OnClose回调函数，连接被关闭后的回调函数
	exitChan := make(chan struct{}, 1)
	c.OnClose(func(conn conn.Conn, err error) (next bool) {
		log.Println("OnClose", conn.RemoteAddr().String())
		exitChan <- struct{}{}
		return true
	})
	// Register 注册实现了handler.Handler的处理接口，该接口的回调函数在OnAccept、OnConnect、OnMessage、OnClose之后被回调
	c.Register(message.MsgType_Default, &handler.Default{
		OnAcceptFunc:  nil,
		OnConnectFunc: nil,
		OnMessageFunc: func(r responsewriter.ResponseWriter, m *message.Message) {
			log.Println("OnMessageFunc handle")
			r.Response(message.StateCode_Success, []byte(fmt.Sprintf("response from %d: ok", c.NodeId())))
		},
		OnCloseFunc: nil,
	})
	err := c.Connect("0.0.0.0:8000")
	if err != nil {
		log.Fatalln(err)
	}
	res, code, err := c.Request(context.Background(), []byte("hello"))
	fmt.Println(code, string(res), err)
	_ = c.Close()
	<-exitChan
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


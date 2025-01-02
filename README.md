# node

## 介绍
node是一个Go（Golang）编写的轻量级TCP框架，node帮助您轻松、快速构建TCP服务器。

特征：
- 支持请求响应模型
- 支持多服务端节点桥接组网
- 支持节点间路由
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
func main() {
	// 创建一个节点Id为8000的节点
	s := node.NewServerOption(8000)
	// 全局注册OnAccept钩子，返回值next是否关闭连接和下一个回调
	s.OnAccept(func(conn net.Conn) (allow bool) {
		log.Println("OnAccept", conn.RemoteAddr().String())
		return true
	})
    // 全局注册 OnConnect钩子，返回值next是否关闭连接和下一个回调
	s.OnConnect(func(conn conn.Conn) (next bool) {
		return true
	})
    // 全局注册 OnMessage，返回值next是否关闭连接和下一个回调
	s.OnMessage(func(r responsewriter.ResponseWriter, m *message.Message) (next bool) {
		r.Response(message.StateCode_Success, []byte(fmt.Sprintf("response from %d: ok", s.NodeId())))
		return true
	})
    // 全局注册 OnClose，返回值next是否关闭连接和下一个回调
	s.OnClose(func(conn conn.Conn, err error) (next bool) {
		return true
	})
	// 注册消息类型为message.MsgType_Default (0) 的 "缺省处理函数"
	s.Register(message.MsgType_Default, &handler.Default{
		OnAcceptFunc:  nil,
		OnConnectFunc: nil,
		OnMessageFunc: func(r responsewriter.ResponseWriter, m *message.Message) {
			log.Println("OnAcceptFunc")
			r.Response(message.StateCode_Success, []byte(fmt.Sprintf("response from %d: ok", s.NodeId())))
		},
		OnCloseFunc: nil,
	})
	// 侦听并启动
	err := s.ListenAndServe("0.0.0.0:8000")
	if err != nil {
		log.Println(err)
	}
}
```

### Client
[Client 完整的示例](example/basic/client/main.go)
```go
func main() {
	// 创建一个节点为8081的节点，服务端节点8000
	c := node.NewClientOption(8081, 8000)
	// 全局注册OnAccept钩子，返回值next是否关闭连接和下一个回调
	c.OnAccept(func(conn net.Conn) (next bool) {
		return true
	})
	// 全局注册 OnConnect钩子，返回值next是否关闭连接和下一个回调
	c.OnConnect(func(conn conn.Conn) (next bool) {
		return true
	})
    // 全局注册 OnMessage，返回值next是否关闭连接和下一个回调
	c.OnMessage(func(r responsewriter.ResponseWriter, m *message.Message) (next bool) {
		if m.Type != message.MsgType_Default {
			r.Response(message.StateCode_MessageTypeInvalid, nil)
			return false
		}
		r.Response(message.StateCode_Success, []byte(fmt.Sprintf("response from %d: ok", c.NodeId())))
		return true
	})
	// 创建连接断开chan
	exitChan := make(chan struct{}, 1)
    // 全局注册 OnMessage，返回值next是否关闭连接和下一个回调
	c.OnClose(func(conn conn.Conn, err error) (next bool) {
		exitChan <- struct{}{}
		return true
	})
	// 连接
	err := c.Connect("0.0.0.0:8000")
	if err != nil {
		log.Fatalln(err)
	}
	// 发起请求，并得到回复
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


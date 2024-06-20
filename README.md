# Node是一个用Golang实现的TCP发送接收封装库，开箱即用

## 概述
node实现了客户端、服务端的封装，基于节点ID、API进行路由处理，提供连接认证、请求、转发、功能，开箱即用，客户端具有路由处理能力，功能上客户端服务端大体相同

代码的各个功能都有相应的注释

## 使用
  go get -u github.com/Li-giegie/node@latest
```go
// 服务端示例 完整代码：example/scene1/server_node
// 开启侦听
srv, err := node.ListenTCP(0, "0.0.0.0:8080")
if err != nil {
    return
}
// 绑定处理函数
srv.HandleFunc(1, func(ctx *common.Context) {
  log.Println("receive: ", ctx.String())
  rData := []byte(fmt.Sprintf("%s reply ", time.Now().Format("2006/01/02 15:04:05")))
  rData = append(rData, ctx.Data()...)
  ctx.Write(rData)
})
// 开启服务
err = srv.Serve()
  if err != nil {
  log.Fatalln(err)
}

// 客户端示例 完整代码：example/scene1/client_node
func main() {
  // 发起连接
  client, err := node.DialTCP(0, "0.0.0.0:8080", 1)
  if err != nil {
      log.Fatal(err)
  }
  // 添加处理方法
  client.HandleFunc(1, func(ctx *common.Context) {
    log.Println("receive", ctx.String())
    ctx.Write([]byte("ok"))
  })
  // 认证连接
  conn, err := client.AuthenticationWithServer(context.Background(), nil)
  if err != nil {
      log.Fatal(err)
  }
  defer conn.Close()
  // 心跳包
  err = conn.Tick(time.Second, time.Second*3, time.Second*10, true)
  if err != nil {
    fmt.Print(err)
    return
  }
  go Request(conn)
  // 开启服务
  err = conn.Serve()
  if err != nil {
    log.Fatal(err)
  }
}

func Request(conn common.Conn) {
  stdin := bufio.NewScanner(os.Stdin)
  fmt.Print(">> ")
  for stdin.Scan() {
    if conn.State() != common.ConnStateTypeOnConnect {
        log.Fatalln("disconnect")
    }
    if len(stdin.Bytes()) == 0 {
      fmt.Print(">> ")
      continue
    }
    log.Println("send: ", stdin.Text())
    resp, err := conn.Request(context.Background(), 1, stdin.Bytes())
    if err != nil {
        log.Fatalln(err)
    }
    fmt.Println(string(resp))
    fmt.Print(">> ")
  }
}
```

## 协议报式
```
Typ        uint8
Id         uint32
SrcId      uint16
DestId     uint16
Api        uint16
DataLength uint32
Data       []byte
```
<table >
  <tr>
    <th rowspan="2" >Header 16Byte</th>
    <td >Typ 1Byte</td>
    <td >Id 3Byte</td>
    <td >SrcId 2Byte</td>
    <td >DestId 2Byte</td>
    <td >Api 2Byte</td>
    <td >DataLength 4Byte</td>
  </tr>
  <tr >
    <td align="center" colspan="7">CheckSum 2Byte</td>
  </tr>
  <tr >
    <td align="center" colspan="7">Data</td>
  </tr>
</table>

## 功能
* 认证：想要与服务节点建立连接首先应该进行认证，与服务端节点建立连接后会立即进入认证环节，认证保证客户端是否为非法。
* 心跳机制：保持连接不会被网络通信设备认为当前连接不活跃被关闭采取的措施，通常是在一定周期内没有收到数据会被激活
* 发送数据
  1. send方法：仅为发送数据，并不需要具有回复，不需要明确目的地，如果api在节点中不存在，并不会通知客户端失败信息，所以客户端并不知晓这一次发送是否真的被处理，如果出现错误只会在tcp连接层出现问题才会产生，不需要明确目的地
  2. request方法：发送数据并希望在等待时间内得到对端的回复，不需要明确目的地
  3. forward方法：发送消息到指定节点处理，需要明确目的ID
## 场景
[客户端节点服务端节点简单请求示例](./example/scene1)

[多客户端节点通过服务端节点互相转发示例](./example/scene1)

## 更新迭代
* 进一步丰富框架功能
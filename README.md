# Node是一个用Golang实现的TCP库

## 概述
node：对连接的功能进行了封装，连接视为节点，每个节点都有一个唯一的ID，功能主要包含有请求、节点间转发数据

代码的主要功能可能有注释

## 协议报式
```go
type Message struct {
  Type   uint8
  Id     uint32
  SrcId  uint16
  DestId uint16
  Data   []byte
}
```
<table >
  <tr>
    <th rowspan="2" >Header 13Byte</th>
    <td >Typ 1Byte</td>
    <td >Id 3Byte</td>
    <td >SrcId 2Byte</td>
    <td >DestId 2Byte</td>
    <td >DataLength 3Byte</td>
    <td >Checksum 2Byte</td>
  </tr>
  <tr >
    <td align="center" colspan="6">CheckSum 2Byte</td>
  </tr>
  <tr >
    <td align="center" colspan="7">Data</td>
  </tr>
</table>

单次数据最大发送长度为3个字节的正整数容量15MB

## 使用
```
go get -u github.com/Li-giegie/node@latest
```
#### Node接口负责连接的生命周期，下文对Node接口进行了介绍，默认接口生命周期是同步调用。
1. Node 一个连接的生命周期
2. Connection 同步调用：该方法在连接建立时被调用，返回对端ID、error，如果err返回值非空，连接关闭，此步骤用于自定义认证，在服务端返回空，且ID唯一（连接列表不存在）连接会被建立，反之服务端会关闭连接并返回连接已存在错误
3. Handle 默认同步调用：异同步取决于你，如果该回调阻塞将影响当前连接整个生命周期（对于一些不消耗时间的任务，重新开启一个goroutine执行未必最优），框架并没有集成协程池，第三方框架众多，一时拿不定主意，索性把问题抛给你
4. ErrHandle 默认同步调用：异同步取决于你，当发送消息失败时会被触发
5. DropHandle 默认同步调用：异同步取决于你，目前仅会在接受消息超时时会被回调
6. CustomHandle 默认同步调用：异同步取决于你，自定义消息类型处理，框架内部集成了多种消息类型，当需要一些特定的功能时可以自定义消息类型，例如心跳消息，只需把消息类型声明成框架内部不存在的类型，框架看到不认识的消息就会回调当前函数
7. Disconnect 同步调用：连接断开会被触发
```go
type Node interface {
	// Connection 连接被建立时触发回调
	Connection(conn net.Conn) (remoteId uint16, err error)
	// Handle 接收到标准类型消息时触发回调
	Handle(ctx *common.Context)
	// ErrHandle 发送失败触发的回调
	ErrHandle(msg *common.Message)
	// DropHandle 接收到超时消息时触发回调
	DropHandle(msg *common.Message)
	// CustomHandle 接收到自定义类型消息时触发回调
	CustomHandle(ctx *common.Context)
	// Disconnect 连接断开触发回调
	Disconnect(id uint16, err error)
}

```
[example示例](example)
## 功能
1. send方法：仅为发送数据，并不需要具有回复，不需要明确目的地，如果api在节点中不存在，并不会通知客户端失败信息，所以客户端并不知晓这一次发送是否真的被处理，如果出现错误只会在tcp连接层出现问题才会产生，不需要明确目的地
2. request方法：发送数据并希望在等待时间内得到对端的回复，不需要明确目的地
3. forward方法：发送消息到指定节点处理，需要明确目的ID

## 更新迭代
* 进一步丰富框架功能
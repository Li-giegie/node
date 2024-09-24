# Node是一个用Golang实现的TCP库

## 概述
node 对连接发送的数据制定了一套协议，基于此协议对连接的常用功能进行了封装，连接视为节点，每个节点都有一个唯一的ID，
主要功能包含有请求、节点间转发数据

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
  </tr>
  <tr >
    <td align="center" colspan="5">CheckSum 2Byte</td>
  </tr>
  <tr >
    <td align="center" colspan="6">Data</td>
  </tr>
</table>

单次数据最大发送长度为3个字节的正整数容量0xFFFFFF(大约15MB)

## 使用
```
go get -u github.com/Li-giegie/node@latest
```
#### Handler接口负责连接的生命周期，下文对Handler接口进行了介绍，默认接口生命周期是同步调用。
```go
type Handler interface {
    Connection(conn common.Conn)
    Handle(ctx common.Context)
    ErrHandle(msg *common.Message, err error)
    CustomHandle(ctx common.Context)
    Disconnect(id uint16, err error)
}
```
1. Connection 连接第一次建立成功回调
2. Handle 接到标准类型消息会被触发，如果该回调阻塞将阻塞当前节点整个生命周期回调（在同步调用模式中如果在这个回调中发起请求需要另外开启协程否则会陷入阻塞，无法接收到消息），框架并没有集成协程池，第三方框架众多，合理选择
3. ErrHandle 当收到：超过限制长度的消息0xffffff、校验和错误、超时、服务节点返回的消息但没有接收的消息都会在这里触发回调
4. CustomHandle 自定义消息类型处理，框架内部默认集成了多种消息类型，当需要一些特定的功能时可以自定义消息类型，例如心跳消息，只需把消息类型声明成框架内部不存在的类型，框架看到不认识的消息就会回调当前函数
5. Disconnect 连接断开会被触发

## 功能
一个域内节点通信示例图如下

![单域](./.README_images/single.png)

多域间节点互相通信如下

![多域](./.README_images/multiple.png)

[example示例](example)

## 协议
[关于协议的进一步使用 README](protocol/README.md)
## 更新迭代
* 增加功能

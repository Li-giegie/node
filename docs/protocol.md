# protocol 是node框架的扩展

```go
type Protocol interface {
    //协议类型
    ProtocolType() uint8 
    // OnAccept accept 行后的回到，allow是否允许接受连接，nil值默认接受
    OnAccept(conn net.Conn) (allow bool)
    // OnConnect 连接通过基础认证正式建立后的回调
    OnConnect(conn conn.Conn)
    // OnMessage 收到消息后的回调
    OnMessage(r responsewriter.ResponseWriter, msg *message.Message)
    // OnClose 连接关闭后的回调
    OnClose(conn conn.Conn, err error)
}
```

# Changelog
- [v1.0](#v10)
- [v1.1](#v11)
- [v1.2](#v12)
- [v1.3](#v13)
___

## v1.0
- 初始版本1.0版本
___

## v1.1
### REFACTOR
- refactor：重构代码实现，修改协议格式，转为轻量化、易扩展、重构认证

### FEAT
- feat：增加节点动态路由发现协议
- feat：增加hello心跳协议模块
- feat：增加protocol\auth json认证模块
- feat：连接直接读取替换为读取缓冲区，增加tick包
### BUG
- fix：修复消息Data消息超过限制大小造成错误
- fix：修复服务端查询路由请求没有修改目的地错误，修改一些代码实现上的风格，protocol模块内输出打印使用默认log库
- fix：修复合并代码导致的转发函数未转发逻辑错误
### DOCS
- docs：提交RADEME文件，示例完善，节点动态路由、认证、心跳协议初步实现完毕
___

## v1.2
### BUG
- fix：修复24位消息id越界问题
### REFACTOR
- refactor：删除消息接收chan缓存，完善example
___

## v1.3
### BUG
- fix：修复服务节点发送消息，子连接Id使用自己的计数器导致从0计数逻辑错误，改为使用父节点全局Id计数器
### CI
- ci：变更一些实现逻辑、更新一些文档
### BENCHMARK
- 文件：test/bench_echo_client_test.go、test/bench_echo_server_test.go
- 函数：TestEchoServer、BenchmarkEchoRequest、BenchmarkEchoRequestGo
- benchmark
```go
go test -run none -bench BenchmarkEchoRequest -benchmem -benchtime 3s -cpu 1
goos: windows
goarch: amd64
pkg: github.com/Li-giegie/node/test
cpu: AMD Ryzen 5 5600H with Radeon Graphics
BenchmarkEchoRequest       70932             50880 ns/op             186 B/op          6 allocs/op
BenchmarkEchoRequestGo    190693             20517 ns/op             808 B/op          8 allocs/op

go test -run none -bench BenchmarkEchoRequest -benchmem -benchtime 3s -cpu 12
goos: windows
goarch: amd64
pkg: github.com/Li-giegie/node/test
cpu: AMD Ryzen 5 5600H with Radeon Graphics
BenchmarkEchoRequest-12            67322             53411 ns/op             186 B/op          6 allocs/op
BenchmarkEchoRequestGo-12         146332             24469 ns/op             743 B/op          8 allocs/op
```
___

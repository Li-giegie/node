# Changelog
- [v1.0](#v10)
- [v1.1](#v11)
- [v1.2](#v12)
- [v1.3](#v13)
- [v1.4](#v14)
- [v1.5](#v15)
- [v2.0](#v20)
- [v2.1](#v21)
- [v3.0](#v30)
- [v3.1](#v31)
- [v3.2](#v32)
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

## v1.4
### FEAT
- feat：增加Writer缓冲区，多核心并发提升50%性能，每秒可达10w次请求响应
### BENCHMARK
- 文件：test/bench_echo_client_test.go、test/bench_echo_server_test.go
- 函数：TestEchoServer、BenchmarkEchoRequest、BenchmarkEchoRequestGo
- benchmark
```go
go test -run none -bench BenchmarkEchoRequest -benchtime 3s -cpu 1
goos: windows
goarch: amd64
pkg: github.com/Li-giegie/node/test
cpu: AMD Ryzen 5 5600H with Radeon Graphics
BenchmarkEchoRequest       69897             51011 ns/op
BenchmarkEchoRequestGo    237777             16545 ns/op

go test -run none -bench BenchmarkEchoRequest -benchmem -benchtime 3s -cpu 12
goos: windows
goarch: amd64
pkg: github.com/Li-giegie/node/test
cpu: AMD Ryzen 5 5600H with Radeon Graphics
BenchmarkEchoRequest-12            66759             53607 ns/op             186 B/op          6 allocs/op
BenchmarkEchoRequestGo-12         304437             12316 ns/op             735 B/op          8 allocs/op
```
___

## v1.5
### CI
- ci：优化一些实现逻辑，更新一些文档
___

## v2.0
### FEAT
- feat：增加write队列，提升性能可达100万并发请求响应
### REFACTOR
- refactor：重构消息结果，id、srcId、destId使用uint32四个字节范围表示
### BENCHMARK
- 文件：test/bench_echo_client_test.go、test/bench_echo_server_test.go
- 函数：TestEchoServer、BenchmarkEchoRequest、BenchmarkEchoRequestGo
- benchmark
```go
go test -run none -bench BenchmarkEchoRequest -benchtime 3s -cpu 1
goos: windows
goarch: amd64
pkg: github.com/Li-giegie/node/test
cpu: AMD Ryzen 5 5600H with Radeon Graphics
BenchmarkEchoRequest       57748             62148 ns/op
BenchmarkEchoRequestGo    698731              6537 ns/op

go test -run none -bench BenchmarkEchoRequest -benchmem -benchtime 3s -cpu 12
goos: windows
goarch: amd64
pkg: github.com/Li-giegie/node/test
cpu: AMD Ryzen 5 5600H with Radeon Graphics
BenchmarkEchoRequest-12            56014             64310 ns/op             186 B/op          6 allocs/op
BenchmarkEchoRequestGo-12        2179160              1808 ns/op             441 B/op          7 allocs/op
```
___

## v2.1
### REFACTOR
- refactor：重构一些代码

## v3.0
### REFACTOR
- refactor：重构

## v3.1
### DOCS
- docs：增加一些注释、示例

## v3.2
### FIX
- fix：修复节点动态发现一些问题和优化实现

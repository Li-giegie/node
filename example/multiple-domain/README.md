## multiple-domain示例展示了多个域之间互联，节点之间简单通信

效果图

![](../../.README_images/multiple.png)

### 构建
```go
cd example/multiple-domain
go build
```
客户端使用example/client/中的客户端
### 启动
```go
domain 10
./multiple-domain -id=10 -laddr=0.0.0.0:8010
./client -lid=1 -raddr 0.0.0.0:8010
./client -lid=2 -raddr 0.0.0.0:8010

domain 20
./multiple-domain -id=20 -laddr=0.0.0.0:8020 -raddr=0.0.0.0:8010 -enablebind
./client -lid=21 -raddr 0.0.0.0:8020

domain 30
./multiple-domain -id=30 -laddr=0.0.0.0:8030 -raddr=0.0.0.0:8020 -enablebind
./client -lid=31 -raddr 0.0.0.0:8030

domain 40
./multiple-domain -id=40 -laddr=0.0.0.0:8040 -raddr=0.0.0.0:8030 -enablebind
./client -lid=41 -raddr 0.0.0.0:8040
./client -lid=42 -raddr 0.0.0.0:8040
```
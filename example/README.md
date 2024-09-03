## 下文将演示图中多节点互联场景

![](../.README_images/multiple.png)

### 指定配置启动服务端节点
节点开启：节点发现（转发节点消息）、hell心跳检测

可以自行编译二进制后指定配置文件启动

启动节点：10、20、30、40
```go
go run ./server/main.go -conf ./conf/srv_conf_10.yaml
go run ./server/main.go -conf ./conf/srv_conf_20.yaml
go run ./server/main.go -conf ./conf/srv_conf_30.yaml
go run ./server/main.go -conf ./conf/srv_conf_40.yaml
```
#### 命令列表 
- help
- bind 绑定一个服务端节点
- list 输出连接信息信息、或节点路由信息
- request 发送消息，并希望在限定时间内得到一个回复
- write 发送数据

更具体的用法每条命令下输入help可查看用法

### 把服务端节点绑定在一起

绑定的顺序为：40 到 30 到 20 到 10
依次在每个服务端节点中执行如下命令，10节点做根节点，根节点不用绑定节点
```
40节点执行
bind -addr 127.0.0.1:8030 -key hello
30节点执行
bind -addr 127.0.0.1:8020 -key hello
20节点执行
bind -addr 127.0.0.1:8010 -key hello
```
绑定成功输出日志：>xxxx/xx/xx xx:xx:xx [DiscoveryNodeProtocol] Connection query enable protocol node id 10
### 指定配置启动客户端节点
节点开启：hell心跳检测
启动节点：1、2、21、31、41、42
```go
go run ./client/main.go -conf ./conf/cli_conf_1.yaml
go run ./client/main.go -conf ./conf/cli_conf_2.yaml
go run ./client/main.go -conf ./conf/cli_conf_21.yaml
go run ./client/main.go -conf ./conf/cli_conf_31.yaml
go run ./client/main.go -conf ./conf/cli_conf_41.yaml
go run ./client/main.go -conf ./conf/cli_conf_42.yaml
```

#### 命令列表
- help 帮助信息
- request 发送数据，并希望再限定时间内得到回复
- write 发送数据

### 命令演示
#### 服务端节点中查看连接
在节点10中执行：list conn
```
0 1
1 2
2 20
```
得到第一个数字为序号、第二个为节点Id，共有三个连接，分别为1、2、20

这里的连接都是直连连接，并不包含所有可以到达的连接，可到达的连接是直连连接加路由可到达的连接

#### 在服务端节点中查看路由
在节点10中执行：list route
```
dest    next    hop     parent-node     time
40      20      2       30              2024-09-04 02:17:50

41      20      3       40              2024-09-04 02:17:50

42      20      3       40              2024-09-04 02:17:50

21      20      1       20              2024-09-04 02:17:38

30      20      1       20              2024-09-04 02:17:38

31      20      2       30              2024-09-04 02:17:50
```
```
目的Id  下一跳Id  跳数    父节点Id         更新时间
```
直连连接加路由就是能够到达的所有节点
#### 发送请求
自己不能给自己发送消息

在节点10上执行
```
request -id 1 hello
```
看到响应内容 hello
```
xxxx/xx/xx xx:xx:xx ClientHandler [1] handle reply: hello
```


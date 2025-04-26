package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/handler"
	"github.com/Li-giegie/node/pkg/message"
	"github.com/Li-giegie/node/pkg/responsewriter"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

func main() {
	id := uint32(8000)
	addr := ":8000"
	// 创建8000服务端节点
	s := node.NewServerOption(id)
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
	// Register 注册实现了handler.Handler的处理接口，该接口的回调函数在全局OnAccept、OnConnect、OnMessage、OnClose之后被回调
	s.Register(message.MsgType_Default, &handler.Default{
		OnMessageFunc: func(r responsewriter.ResponseWriter, m *message.Message) {
			log.Println("Register OnMessage", m.String())
			r.Response(message.StateCode_Success, append([]byte(fmt.Sprintf("response from %d: ", s.NodeId())), m.Data...))
		},
	})
	go func() {
		scan := bufio.NewScanner(os.Stdin)
		print(">>")
		for scan.Scan() {
			fields := strings.Fields(scan.Text())
			if len(fields) > 0 {
				switch fields[0] {
				case "bridge":
					if len(fields) < 3 {
						println("invalid command", scan.Text())
					} else {
						c, err := net.Dial("tcp", fields[1])
						if err != nil {
							println("dial error", err.Error())
						} else {
							rid, err := strconv.Atoi(fields[2])
							if err != nil {
								println("invalid remote id", fields[2])
							} else {
								fmt.Println(fields)
								key := []byte{}
								if len(fields) > 3 {
									key = []byte(fields[3])
								}
								err = s.Bridge(c, uint32(rid), key)
								if err != nil {
									println("bridge error", err.Error())
								}
							}
						}
					}
				case "exit":
					s.Close()
					println("bye~")
					return
				case "help":
					println("bridge usage: bridge [remote address] [remote id] [remote key]")
					println("request usage: request [remote id] [data]")
					println("exit usage: exit")
				case "request":
					if len(fields) != 3 {
						println("invalid command", scan.Text())
					} else {
						rid, err := strconv.Atoi(fields[1])
						if err != nil {
							println("invalid remote id", fields[1])
						} else {
							resp, code, err := s.RequestTo(context.TODO(), uint32(rid), []byte(fields[2]))
							if err != nil {
								println(err.Error())
							} else {
								println(code, string(resp))
							}
						}
					}
				default:
					for _, c := range s.GetAllConn() {
						res, code, err := c.Request(context.Background(), scan.Bytes())
						if err != nil {
							fmt.Println(err)
						} else {
							fmt.Println(code, string(res))
						}
					}
				}
			}
			print(">>")
		}
	}()
	// 侦听并开启服务
	err := s.ListenAndServe(addr)
	if err != nil {
		log.Println(err)
	}
}

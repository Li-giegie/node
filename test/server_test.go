package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/protocol"
	"log"
	"net"
	"testing"
	"time"
)

func TestServer1(t *testing.T) {
	err := StartServer("0.0.0.0:8001", 1, func(srv iface.Server) {
		go func() {
			for {
				time.Sleep(time.Second)
				fmt.Println("conns", srv.GetAllId())
				srv.RangeRoute(func(id uint64, dst uint32, via uint32, hop uint8) {
					fmt.Printf("route id %d dst %d via %d hop %d\n", id, dst, via, hop)
				})
			}
		}()
	})
	if err != nil {
		log.Println(err)
	}
}

func TestServer2(t *testing.T) {
	err := StartServer("0.0.0.0:8002", 2, func(srv iface.Server) {
		go func() {
			time.Sleep(time.Second)
			fmt.Println("开始绑定")
			conn, _ := net.Dial("tcp", "0.0.0.0:8001")
			fmt.Println(srv.Bridge(conn, []byte("hello"), time.Second))
		}()
		go func() {
			for {
				time.Sleep(time.Second)
				fmt.Println("conns", srv.GetAllId())
				srv.RangeRoute(func(id uint64, dst uint32, via uint32, hop uint8) {
					fmt.Printf("route id %d dst %d via %d hop %d\n", id, dst, via, hop)
				})
			}
		}()
	})
	if err != nil {
		log.Println(err)
	}
}

func TestServer3(t *testing.T) {
	err := StartServer("0.0.0.0:8003", 3, func(srv iface.Server) {
		go func() {
			time.Sleep(time.Second)
			fmt.Println("开始绑定")
			conn, _ := net.Dial("tcp", "0.0.0.0:8001")
			fmt.Println(srv.Bridge(conn, []byte("hello"), time.Second))
			conn2, _ := net.Dial("tcp", "0.0.0.0:8002")
			fmt.Println(srv.Bridge(conn2, []byte("hello"), time.Second))
		}()
		go func() {
			for {
				time.Sleep(time.Second)
				fmt.Println("conns", srv.GetAllId())
				srv.RangeRoute(func(id uint64, dst uint32, via uint32, hop uint8) {
					fmt.Printf("route id %d dst %d via %d hop %d\n", id, dst, via, hop)
				})
			}
		}()
	})
	if err != nil {
		log.Println(err)
	}
}

func TestServer4(t *testing.T) {
	err := StartServer("0.0.0.0:8004", 4, func(srv iface.Server) {
		go func() {
			time.Sleep(time.Second)
			fmt.Println("开始绑定")
			conn1, _ := net.Dial("tcp", "0.0.0.0:8001")
			fmt.Println(srv.Bridge(conn1, []byte("hello"), time.Second*3))
			conn2, _ := net.Dial("tcp", "0.0.0.0:8002")
			fmt.Println(srv.Bridge(conn2, []byte("hello"), time.Second*3))
			conn3, _ := net.Dial("tcp", "0.0.0.0:8003")
			fmt.Println(srv.Bridge(conn3, []byte("hello"), time.Second*3))
		}()
		go func() {
			for {
				time.Sleep(time.Second)
				fmt.Println("conns", srv.GetAllId())
				srv.RangeRoute(func(id uint64, dst uint32, via uint32, hop uint8) {
					fmt.Printf("route id %d dst %d via %d hop %d\n", id, dst, via, hop)
				})
			}
		}()
	})
	if err != nil {
		log.Println(err)
	}
}

func TestServer5(t *testing.T) {
	err := StartServer("0.0.0.0:8005", 5, func(srv iface.Server) {
		go func() {
			time.Sleep(time.Second)
			fmt.Println("开始绑定")
			conn1, _ := net.Dial("tcp", "0.0.0.0:8001")
			fmt.Println(srv.Bridge(conn1, []byte("hello"), time.Second))
			conn3, _ := net.Dial("tcp", "0.0.0.0:8003")
			fmt.Println(srv.Bridge(conn3, []byte("hello"), time.Second))
			//conn4, _ := net.Dial("tcp", "0.0.0.0:8004")
			//fmt.Println(srv.Bridge(conn4, []byte("hello"), time.Second))
		}()
		go func() {
			for {
				time.Sleep(time.Second)
				fmt.Println("conns", srv.GetAllId())
				srv.RangeRoute(func(id uint64, dst uint32, via uint32, hop uint8) {
					fmt.Printf("route id %d dst %d via %d hop %d\n", id, dst, via, hop)
				})
			}
		}()
	})
	if err != nil {
		log.Println(err)
	}
}

func StartServer(addr string, id uint32, f func(s iface.Server)) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := node.NewServer(l, &node.SrvConf{
		Identity: &node.Identity{
			Id:          id,
			AuthKey:     []byte("hello"),
			AuthTimeout: time.Second * 6,
		},
		MaxMsgLen:          0xffffff,
		WriterQueueSize:    1024,
		ReaderBufSize:      4096,
		WriterBufSize:      4096,
		MaxConns:           0,
		MaxListenSleepTime: time.Minute,
		ListenStepTime:     time.Second,
	})
	protocol.StartDiscoveryProtocol(16, srv, srv)
	protocol.StartMultipleNodeHelloProtocol(context.Background(), srv, srv, time.Minute, time.Minute, time.Minute*4, nil)
	srv.AddOnConnection(func(conn iface.Conn) {
		log.Println("OnConnection", conn.RemoteId(), conn.NodeType())
	})
	srv.AddOnMessage(func(ctx iface.Context) {
		log.Println("OnMessage", ctx.String())
		ctx.Reply(ctx.Data())
	})
	srv.AddOnCustomMessage(func(ctx iface.Context) {
		log.Println("OnCustomMessage", ctx.String())
	})
	srv.AddOnClosed(func(conn iface.Conn, err error) {
		log.Println(conn.RemoteId(), err, conn.NodeType())
	})
	go f(srv)
	defer srv.Close()
	return srv.Serve()
}

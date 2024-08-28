package cmd

import (
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/protocol"
	rabbit "github.com/Li-giegie/rabbit-cli"
	"log"
	"net"
	"time"
)

var bind = &rabbit.Cmd{
	Name:        "bind",
	Description: "创建一个客户端连接，并绑定",
	RunE: func(c *rabbit.Cmd, args []string) error {
		key := c.Flag().Lookup("key").Value.String()
		addr := c.Flag().Lookup("addr").Value.String()
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return err
		}
		i := c.Context().Value("server")
		if i == nil {
			return fmt.Errorf("server is null")
		}
		srv := i.(node.Server)
		eNode, err := node.NewExternalDomainNode(conn, func(conn net.Conn) (remoteId uint16, err error) {
			return protocol.NewClientAuthProtocol(srv.Id(), key, time.Second*6).Init(conn)
		})
		if err != nil {
			return err
		}
		go func() {
			if err = srv.Bind(eNode); err != nil {
				log.Println("bind node disconnected")
			}
		}()
		return nil
	},
}

func init() {
	bind.Flag().String("key", "hello", "remote key")
	bind.Flag().String("addr", "", "remote addr")
	bind.AddSubMust(&rabbit.Cmd{
		Name:        "help",
		Description: "帮助信息",
		Run: func(c *rabbit.Cmd, args []string) {
			bind.Usage()
		},
		RunE: nil,
	})
	Group.AddCmdMust(bind)
}

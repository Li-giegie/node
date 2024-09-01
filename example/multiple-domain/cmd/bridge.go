package cmd

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	rabbit "github.com/Li-giegie/rabbit-cli"
	"log"
	"net"
	"strconv"
	"time"
)

var bind = &rabbit.Cmd{
	Name:        "bind",
	Description: "创建一个客户端连接，并绑定",
	RunE: func(c *rabbit.Cmd, args []string) error {
		rid, _ := strconv.Atoi(c.Flags().Lookup("id").Value.String())
		addr := c.Flags().Lookup("addr").Value.String()
		timeout, _ := time.ParseDuration(c.Flags().Lookup("timeout").Value.String())
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return err
		}
		i := c.Context().Value("server")
		if i == nil {
			return fmt.Errorf("server is null")
		}
		srv := i.(*node.Server)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		bn, err := node.CreateBridgeNode(ctx, conn, srv.Id(), uint16(rid), func() {
			log.Println("桥接节点断开连接", rid)
		})
		if err != nil {
			_ = conn.Close()
			return err
		}
		return srv.BindBridge(bn)
	},
}

func init() {
	bind.Flags().Uint("id", 0, "remote id")
	bind.Flags().String("addr", "", "remote addr")
	bind.Flags().Duration("timeout", time.Second*3, "init timeout")
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

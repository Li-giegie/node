package cmd

import (
	"errors"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/iface"
	rabbit "github.com/Li-giegie/rabbit-cli"
)

var list = &rabbit.Cmd{
	Name:        "list",
	Description: "输出信息",
	Run: func(c *rabbit.Cmd, args []string) {
		c.Usage()
	},
}

func init() {
	list.AddSubMust(&rabbit.Cmd{
		Name:        "conn",
		Description: "输出连接数",
		RunE: func(c *rabbit.Cmd, args []string) error {
			i := c.Context().Value("server")
			if i == nil {
				return errors.New("server is null")
			}
			srv := i.(*node.Server)
			for i2, conn := range srv.ConnManager.GetAll() {
				fmt.Println(i2, conn.RemoteId())
			}
			return nil
		},
	})
	list.AddSubMust(&rabbit.Cmd{
		Name:        "route",
		Description: "",
		Run:         nil,
		RunE: func(c *rabbit.Cmd, args []string) error {
			i := c.Context().Value("server")
			if i == nil {
				return errors.New("server is null")
			}
			i.(iface.Server).RangeRoute(func(id uint64, dst uint32, via uint32, hop uint8) {
				fmt.Printf("id %d dst %d via %d hop %d\n", id, dst, via, hop)
			})
			return nil
		},
	})
	Group.AddCmd(list)
}

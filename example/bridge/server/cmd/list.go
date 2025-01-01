package cmd

import (
	"errors"
	"fmt"
	"github.com/Li-giegie/node/pkg/router"
	"github.com/Li-giegie/node/pkg/server"
	rabbit "github.com/Li-giegie/rabbit-cli"
	"time"
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
			srv := c.Context().Value("server").(server.Server)
			if srv == nil {
				return errors.New("server is null")
			}
			for i2, conn := range srv.GetAllConn() {
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
			srv := c.Context().Value("server").(server.Server)
			srv.GetRouter().RangeRoute(func(empty *router.RouteEmpty) bool {
				fmt.Print("dst ", empty.Dst, " via ", empty.Via, " date-time ", time.UnixMicro(time.Duration(empty.UnixNano).Microseconds()).Format("2006-01-02 15:04:05"), " hop ", empty.Hop)
				fmt.Print(" path ")
				for _, path := range empty.Paths {
					fmt.Print(path.Id, " ")
				}
				fmt.Println()
				return true
			})
			return nil
		},
	})
	Group.AddCmd(list)
}

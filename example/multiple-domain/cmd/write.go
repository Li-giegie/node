package cmd

import (
	"fmt"
	"github.com/Li-giegie/node"
	rabbit "github.com/Li-giegie/rabbit-cli"
	"strconv"
	"strings"
)

var write = &rabbit.Cmd{
	Name:        "write",
	Description: "发送数据",
	Run:         nil,
	RunE: func(c *rabbit.Cmd, args []string) error {
		i := c.Context().Value("server")
		if i == nil {
			return fmt.Errorf("server is null")
		}
		id, err := strconv.Atoi(c.Flag().Lookup("id").Value.String())
		if err != nil {
			return err
		}
		srv := i.(*node.Server)
		_, err = srv.WriteTo(uint16(id), []byte(strings.Join(args, " ")))
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	write.Flag().Uint("id", 0, "id")
	write.AddSubMust(&rabbit.Cmd{
		Name:        "help",
		Description: "帮助信息",
		Run: func(c *rabbit.Cmd, args []string) {
			write.Usage()
		},
	})
	Group.AddCmdMust(write)
}

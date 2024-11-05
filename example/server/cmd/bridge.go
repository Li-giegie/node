package cmd

import (
	"github.com/Li-giegie/node/iface"
	rabbit "github.com/Li-giegie/rabbit-cli"
	"net"
	"time"
)

var bind = &rabbit.Cmd{
	Name:        "bind",
	Description: "绑定一个服务端节点",
	RunE: func(c *rabbit.Cmd, args []string) error {
		key := c.Flags().Lookup("key").Value.String()
		addr := c.Flags().Lookup("addr").Value.String()
		timeout, _ := time.ParseDuration(c.Flags().Lookup("timeout").Value.String())
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return err
		}
		s := c.Context().Value("server").(iface.Server)
		_, err = s.Bridge(conn, []byte(key), timeout)
		return err
	},
}

func init() {
	bind.Flags().String("key", "", "remote AccessKey")
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

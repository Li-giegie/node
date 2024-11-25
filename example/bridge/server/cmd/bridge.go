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
		var errs [4]error
		var id uint32
		var key, addr string
		var timeout time.Duration
		id, errs[0] = c.Flags().GetUint32("id")
		key, errs[1] = c.Flags().GetString("key")
		addr, errs[2] = c.Flags().GetString("addr")
		timeout, errs[3] = c.Flags().GetDuration("timeout")
		for _, err := range errs {
			if err != nil {
				return err
			}
		}
		srv := c.Context().Value("server").(iface.Server)
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return err
		}
		return srv.Bridge(conn, id, []byte(key), timeout)
	},
}

func init() {
	bind.Flags().Uint("id", 0, "remoteId")
	bind.Flags().String("key", "hello", "remote AccessKey")
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

package cmd

import (
	"github.com/Li-giegie/node/iface"
	rabbit "github.com/Li-giegie/rabbit-cli"
	"strings"
)

var send = &rabbit.Cmd{
	Name:        "send",
	Description: "发送数据",
	Run:         nil,
	RunE: func(c *rabbit.Cmd, args []string) error {
		srv := c.Context().Value("server").(iface.Server)
		id, err := c.Flags().GetUint32("id")
		if err != nil {
			return err
		}
		return srv.SendTo(id, []byte(strings.Join(args, " ")))
	},
}

func init() {
	send.Flags().Uint("id", 0, "id")
	send.AddSubMust(&rabbit.Cmd{
		Name:        "help",
		Description: "帮助信息",
		Run: func(c *rabbit.Cmd, args []string) {
			send.Usage()
		},
	})
	Group.AddCmdMust(send)
}

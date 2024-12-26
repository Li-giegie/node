package cmd

import (
	"github.com/Li-giegie/node/iface"
	rabbit "github.com/Li-giegie/rabbit-cli"
	"strconv"
	"strings"
)

var send = &rabbit.Cmd{
	Name:        "send",
	Description: "发送数据",
	Run:         nil,
	RunE: func(c *rabbit.Cmd, args []string) error {
		id, _ := strconv.Atoi(c.Flags().Lookup("id").Value.String())
		conn := c.Context().Value("conn").(iface.Conn)
		err := conn.SendTo(uint32(id), []byte(strings.Join(args, " ")))
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	send.Flags().Uint("id", 0, "remote id")
	send.AddSubMust(&rabbit.Cmd{
		Name:        "help",
		Description: "write 帮助信息",
		Run: func(c *rabbit.Cmd, args []string) {
			send.Usage()
		},
	})
	Group.AddCmdMust(send)
}

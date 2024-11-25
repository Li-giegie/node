package cmd

import (
	"github.com/Li-giegie/node/iface"
	rabbit "github.com/Li-giegie/rabbit-cli"
	"strconv"
	"strings"
)

var write = &rabbit.Cmd{
	Name:        "write",
	Description: "发送数据",
	Run:         nil,
	RunE: func(c *rabbit.Cmd, args []string) error {
		id, _ := strconv.Atoi(c.Flags().Lookup("id").Value.String())
		conn := c.Context().Value("conn").(iface.Conn)
		_, err := conn.WriteTo(uint32(id), []byte(strings.Join(args, " ")))
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	write.Flags().Uint("id", 0, "remote id")
	write.AddSubMust(&rabbit.Cmd{
		Name:        "help",
		Description: "write 帮助信息",
		Run: func(c *rabbit.Cmd, args []string) {
			write.Usage()
		},
	})
	Group.AddCmdMust(write)
}

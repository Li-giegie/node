package cmd

import (
	rabbit "github.com/Li-giegie/rabbit-cli"
	"strings"
)

var envname = &rabbit.Cmd{
	Name:        "setname",
	Description: "设置环境名称",
	Run: func(c *rabbit.Cmd, args []string) {
		if len(args) == 0 {
			println("无效名称")
			return
		}
		env_name := c.Context().Value("env_name").(*string)
		*env_name = strings.Join(args, " ")
	},
	RunE: nil,
}

func init() {
	Group.AddCmdMust(envname)
}

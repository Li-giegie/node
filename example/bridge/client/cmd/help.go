package cmd

import (
	rabbit "github.com/Li-giegie/rabbit-cli"
)

var help = &rabbit.Cmd{
	Name:        "help",
	Description: "帮助信息",
	Run: func(c *rabbit.Cmd, args []string) {
		Group.Usage()
	},
}

func init() {
	Group.AddCmdMust(help)
}

package cmd

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node/pkg/server"
	rabbit "github.com/Li-giegie/rabbit-cli"
	"strings"
	"time"
)

var request = &rabbit.Cmd{
	Name:        "request",
	Description: "发送消息，并希望在限定时间内得到一个回复",
	Run:         nil,
	RunE: func(c *rabbit.Cmd, args []string) error {
		srv := c.Context().Value("server").(server.Server)
		id, err := c.Flags().GetUint32("id")
		if err != nil {
			return err
		}
		timeout, err := c.Flags().GetDuration("timeout")
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		resp, stateCode, err := srv.RequestTo(ctx, id, []byte(strings.Join(args, " ")))
		if err != nil {
			fmt.Println(err)
			return err
		}
		fmt.Println(stateCode, string(resp))
		return nil
	},
}

func init() {
	request.AddSubMust(&rabbit.Cmd{
		Name:        "help",
		Description: "帮助信息",
		Run: func(c *rabbit.Cmd, args []string) {
			request.Usage()
		},
		RunE: nil,
	})
	request.Flags().Uint("id", 0, "目的id")
	request.Flags().Duration("timeout", time.Second*3, "超时时间")
	Group.AddCmdMust(request)
}

package cmd

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	rabbit "github.com/Li-giegie/rabbit-cli"
	"log"
	"strconv"
	"strings"
	"time"
)

var request = &rabbit.Cmd{
	Name:        "request",
	Description: "发送消息，并希望在限定时间内得到一个回复",
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
		timeout, err := time.ParseDuration(c.Flag().Lookup("timeout").Value.String())
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		srv := i.(node.Server)
		res, err := srv.Request(ctx, uint16(id), []byte(strings.Join(args, " ")))
		if err != nil {
			return err
		}
		log.Println(string(res))
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
	request.Flag().Uint("id", 0, "目的id")
	request.Flag().Duration("timeout", time.Second*3, "超时时间")
	Group.AddCmdMust(request)
}

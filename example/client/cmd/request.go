package cmd

import (
	"context"
	"github.com/Li-giegie/node/common"
	rabbit "github.com/Li-giegie/rabbit-cli"
	"log"
	"strconv"
	"strings"
	"time"
)

var request = &rabbit.Cmd{
	Name:        "request",
	Description: "发送数据，并希望再限定时间内得到回复",
	Run:         nil,
	RunE: func(c *rabbit.Cmd, args []string) error {
		id, _ := strconv.Atoi(c.Flags().Lookup("id").Value.String())
		timeout, _ := time.ParseDuration(c.Flags().Lookup("timeout").Value.String())

		ctx, cancle := context.WithTimeout(context.Background(), timeout)
		defer cancle()
		conn := c.Context().Value("client").(common.Conn)
		res, err := conn.Forward(ctx, uint32(id), []byte(strings.Join(args, " ")))
		if err != nil {
			return err
		}
		log.Println(string(res))
		return nil
	},
}

func init() {
	request.Flags().Uint("id", 0, "remote id")
	request.Flags().Duration("timeout", time.Second*3, "请求超时时间")
	request.AddSubMust(&rabbit.Cmd{
		Name:        "help",
		Description: "request 帮助信息",
		Run: func(c *rabbit.Cmd, args []string) {
			request.Usage()
		},
	})
	Group.AddCmdMust(request)
}

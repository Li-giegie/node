package cmd

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/protocol/nodediscovery"
	rabbit "github.com/Li-giegie/rabbit-cli"
	"strconv"
	"strings"
	"time"
)

var request = &rabbit.Cmd{
	Name:        "request",
	Description: "发送消息，并希望在限定时间内得到一个回复",
	Run:         nil,
	RunE: func(c *rabbit.Cmd, args []string) error {
		srv := c.Context().Value("server").(iface.Server)
		ndp := c.Context().Value("ndp").(nodediscovery.NodeDiscoveryProtocol)
		id, err := strconv.Atoi(c.Flags().Lookup("id").Value.String())
		if err != nil {
			return err
		}
		timeout, err := time.ParseDuration(c.Flags().Lookup("timeout").Value.String())
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		empty, ok := ndp.GetRoute(uint32(id))
		if !ok {
			return fmt.Errorf("%d node not exist", id)
		}
		conn, ok := srv.GetConn(empty.Via())
		if !ok {
			return fmt.Errorf("%d node not exist", id)
		}
		res, err := conn.RequestTo(ctx, uint32(id), []byte(strings.Join(args, " ")))
		if err != nil {
			return err
		}
		fmt.Println(string(res))
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

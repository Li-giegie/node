package cmd

import (
	"fmt"
	"github.com/Li-giegie/node/iface"
	"github.com/Li-giegie/node/protocol/nodediscovery"
	rabbit "github.com/Li-giegie/rabbit-cli"
	"strconv"
	"strings"
)

var write = &rabbit.Cmd{
	Name:        "write",
	Description: "发送数据",
	Run:         nil,
	RunE: func(c *rabbit.Cmd, args []string) error {
		srv := c.Context().Value("server").(iface.Server)
		ndp := c.Context().Value("ndp").(nodediscovery.NodeDiscoveryProtocol)
		id, err := strconv.Atoi(c.Flags().Lookup("id").Value.String())
		if err != nil {
			return err
		}
		conn, ok := srv.GetConn(uint32(id))
		if ok {
			_, err = conn.Write([]byte(strings.Join(args, " ")))
			return err
		}
		empty, ok := ndp.GetRoute(uint32(id))
		if !ok {
			return fmt.Errorf("%d node not exist", id)
		}
		conn, ok = srv.GetConn(empty.Via())
		if !ok {
			return fmt.Errorf("%d node not exist", id)
		}
		_, err = conn.WriteTo(uint32(id), []byte(strings.Join(args, " ")))
		return err
	},
}

func init() {
	write.Flags().Uint("id", 0, "id")
	write.AddSubMust(&rabbit.Cmd{
		Name:        "help",
		Description: "帮助信息",
		Run: func(c *rabbit.Cmd, args []string) {
			write.Usage()
		},
	})
	Group.AddCmdMust(write)
}

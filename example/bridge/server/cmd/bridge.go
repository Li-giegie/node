package cmd

import (
	"github.com/Li-giegie/node/iface"
	rabbit "github.com/Li-giegie/rabbit-cli"
	"log"
	"net"
	"time"
)

var bind = &rabbit.Cmd{
	Name:        "bind",
	Description: "绑定一个服务端节点",
	RunE: func(c *rabbit.Cmd, args []string) error {
		key := c.Flags().Lookup("key").Value.String()
		addr := c.Flags().Lookup("addr").Value.String()
		timeout, _ := time.ParseDuration(c.Flags().Lookup("timeout").Value.String())
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return err
		}
		s := c.Context().Value("server").(iface.Server)
		rid, err := s.Bridge(conn, []byte(key), timeout)
		s.AddOnClosed(func(conn iface.Conn, err error) {
			if conn.RemoteId() != rid {
				return
			}
			go func() {
				log.Println("bridge closed", err)
				for {
					time.Sleep(time.Second * 3)
					conn, err := net.Dial("tcp", addr)
					if err != nil {
						log.Println("dial err", err)
						continue
					}
					rid, err = s.Bridge(conn, []byte(key), timeout)
					if err != nil {
						log.Println("bridge err", err)
					}
					log.Println("bridge success", rid)
					return
				}
			}()
		})
		return err
	},
}

func init() {
	bind.Flags().String("key", "hello", "remote AccessKey")
	bind.Flags().String("addr", "", "remote addr")
	bind.Flags().Duration("timeout", time.Second*3, "init timeout")
	bind.AddSubMust(&rabbit.Cmd{
		Name:        "help",
		Description: "帮助信息",
		Run: func(c *rabbit.Cmd, args []string) {
			bind.Usage()
		},
		RunE: nil,
	})
	Group.AddCmdMust(bind)
}

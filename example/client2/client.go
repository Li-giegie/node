package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"log"
	"os"
	"strings"
	"time"
)

const (
	clientReqApi  = 100
	clientSendApi = 101
)
const serverAddr = "39.101.193.248:8088"

var srvaddr = flag.String("rip", serverAddr, "remote ip")

type CmdList struct {
	top1 *flag.FlagSet
	send *bool
	req  *bool
	exit *bool
	hapi *int
}

func (c *CmdList) Parse(s string) (send, req, exit bool, api int, err error) {
	if err = c.top1.Parse(strings.Split(s, " ")); err != nil {
		return
	}
	return *c.send, *c.req, *c.exit, *c.hapi, nil
}

func newCmdList() *CmdList {
	cl := new(CmdList)
	cl.top1 = flag.NewFlagSet("model", flag.ContinueOnError)
	cl.send = cl.top1.Bool("send", false, "send view")
	cl.req = cl.top1.Bool("req", false, "request view")
	cl.exit = cl.top1.Bool("exit", false, "exit view")
	cl.hapi = cl.top1.Int("api", 0, "handle api ")
	return cl
}

func main() {
	flag.Parse()
	Client2("0.0.0.0:8899", *srvaddr, 2)
}

func Client2(lAddr, rAddr string, id uint64) {
	client := node.NewClient(
		rAddr,
		node.WithClientId(id),
		node.WithClientLocalIpAddr(lAddr),
		node.WithClientKeepAlive(time.Second*3),
	)
	_, err := client.Connect(1, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer client.Close(true)
	fmt.Println("cmd : [send] | [request] | [exit]")
	cmds := newCmdList()
	for {
		str, err := input(">> ")
		if err != nil {
			fmt.Println(err)
			return
		}
		send, req, exit, hapi, err := cmds.Parse(str)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if !send && !req && !exit {
			fmt.Printf("未知命令：%v\n", str)
			continue
		}
		if exit {
			fmt.Println("Bye ~")
			return
		}
	l1:
		for {
			m := "send"
			if req {
				m = "req"
			}
			str2, err2 := input(fmt.Sprintf("%s %d# ", m, hapi))
			if err2 != nil {
				fmt.Println(err2)
				continue
			}
			switch str2 {
			case "q", "quit", "exit":
				break l1
			default:
				if send {
					if err = client.Send(uint32(hapi), []byte(str2)); err != nil {
						fmt.Println("send err: ", err)
						break
					}
					fmt.Println("send# send success")
				} else {
					h := time.Now()
					reply, err := client.Request(time.Second*3, uint32(hapi), []byte(str2))
					if err != nil {
						log.Println("request# request err: ", err, time.Since(h))
						continue
					}
					log.Println("request# request success: ", string(reply), time.Since(h))
				}
			}
		}
	}
}

func input(tag ...string) (string, error) {
	if len(tag) == 0 {
		tag = []string{">> "}
	}
	fmt.Print(tag[0])
	r := bufio.NewReader(os.Stdin)
	str, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	str = strings.Replace(strings.Replace(str, "\r", "", 1), "\n", "", 1)
	return str, nil
}

package main

import (
	"bufio"
<<<<<<< HEAD
	"context"
=======
>>>>>>> dev231223
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

<<<<<<< HEAD
var srvaddr = flag.String("rip", "127.0.0.1:8088", "remote ip")

func main() {
	flag.Parse()

=======
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
>>>>>>> dev231223
	Client2("0.0.0.0:8899", *srvaddr, 2)
}

func Client2(lAddr, rAddr string, id uint64) {
	client := node.NewClient(
		rAddr,
		node.WithClientId(id),
		node.WithClientLocalIpAddr(lAddr),
		node.WithClientKeepAlive(time.Second*3),
	)
<<<<<<< HEAD
	_, err := client.Connect()
=======
	_, err := client.Connect(1, nil)
>>>>>>> dev231223
	if err != nil {
		log.Println(err)
		return
	}
	defer client.Close(true)
	fmt.Println("cmd : [send] | [request] | [exit]")
<<<<<<< HEAD
	for {

=======
	cmds := newCmdList()
	for {
>>>>>>> dev231223
		str, err := input(">> ")
		if err != nil {
			fmt.Println(err)
			return
		}
<<<<<<< HEAD
	top1:
		switch str {
		case "send":
			fmt.Println("cmd: [q|quit] | send text")
			for {
				send, err := input("send# ")
				if err != nil {
					fmt.Println("send# input err: ", err)
					continue
				}
				switch send {
				case "quit", "q":
					break top1
				case "":
					continue
				default:
					if err = client.Send(clientSendApi, []byte(send)); err != nil {
=======
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
>>>>>>> dev231223
						fmt.Println("send err: ", err)
						break
					}
					fmt.Println("send# send success")
<<<<<<< HEAD
				}
			}
		case "request", "req":
			for {
				request, err := input("request# ")
				if err != nil {
					fmt.Println("request# input err: ", err)
					continue
				}
				switch request {
				case "quit", "q":
					break top1
				case "":
					continue
				default:
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
					h := time.Now()
					reply, err := client.Request(ctx, clientReqApi, []byte(request))
					if err != nil {
						log.Println("request# request err: ", err, time.Since(h))
						cancel()
						break
					}
					log.Println("request# request success: ", string(reply), time.Since(h))
					cancel()
				}
			}
		case "exit":
			return
		default:
			fmt.Println("invalid cmd: ", str)
		}

=======
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
>>>>>>> dev231223
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

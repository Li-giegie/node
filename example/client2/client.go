package main

import (
	"bufio"
	"context"
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

var srvaddr = flag.String("rip", "127.0.0.1:8088", "remote ip")

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
	_, err := client.Connect()
	if err != nil {
		log.Println(err)
		return
	}
	defer client.Close(true)
	fmt.Println("cmd : [send] | [request] | [exit]")
	for {

		str, err := input(">> ")
		if err != nil {
			fmt.Println(err)
			return
		}
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
						fmt.Println("send err: ", err)
						break
					}
					fmt.Println("send# send success")
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

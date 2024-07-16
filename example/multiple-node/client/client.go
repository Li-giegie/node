package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/protocol"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type ClientHandle struct {
	id   uint16
	addr string
	protocol.ClientAuthProtocol
	common.Conn
}

func (c *ClientHandle) Init(conn net.Conn) (remoteId uint16, err error) {
	return c.ClientAuthProtocol.Init(conn)
}

func (c *ClientHandle) Connection(conn common.Conn) {
	log.Println("connection", conn.RemoteId())
}

func (c *ClientHandle) Handle(ctx common.Context) {
	log.Println("Handle", ctx.String())
	ctx.Reply([]byte(fmt.Sprintf("client [%d] reply: %s", c.id, ctx.Data())))
}

func (c *ClientHandle) ErrHandle(msg *common.Message, err error) {
	log.Println("ErrHandle", msg.String(), err)
}

func (c *ClientHandle) CustomHandle(ctx common.Context) {
	log.Println("CustomHandle", ctx.String())
}

func (c *ClientHandle) Disconnect(id uint16, err error) {
	log.Println("Disconnect", id, err)
}

func (c *ClientHandle) Serve() error {
	conn, err := node.Dial("tcp", c.addr, c.id, c)
	if err != nil {
		return err
	}

	c.Conn = conn
	return nil
}

func NewClientHandle(id uint16, addr string, key string) *ClientHandle {
	c := new(ClientHandle)
	c.id = id
	c.addr = addr
	c.ClientAuthProtocol = protocol.NewClientAuthProtocol(c.id, key, time.Second*6)
	return c
}

var raddr = flag.String("raddr", "0.0.0.0:8000", "remote address")
var key = flag.String("key", "hello", "key")
var id = flag.Uint("id", 1, "id")

func main() {
	flag.Parse()
	fs := initFlagSet()
	c := NewClientHandle(uint16(*id), *raddr, *key)
	if err := c.Serve(); err != nil {
		log.Fatalln("serve", err)
	}
	defer c.Close()
	scan := bufio.NewScanner(os.Stdin)
	fmt.Print(">>")
	for scan.Scan() {
		switch scan.Text() {
		case "":
		case "exit", "q":
			fmt.Println("bye ~")
			return
		default:
			err := fs.Parse(strings.Split(scan.Text(), " "))
			if err != nil {
				if err.Error() == "flag: help requested" {
					break
				}
				log.Println(err)
				return
			}
			id := fs.Lookup("id")
			t := fs.Lookup("t")
			typ := fs.Lookup("type")
			data := []byte(strings.Join(fs.Args(), " "))
			if len(data) == 0 {
				break
			}
			switch typ.Value.String() {
			case "send":
				if err = c.Send(data); err != nil {
					log.Println("send err", err)
					return
				}
			case "request":
				timeout, err := strconv.Atoi(t.Value.String())
				if err != nil {
					log.Println("t (timeout) value invalid")
					break
				}
				remoteId, err := strconv.Atoi(id.Value.String())
				if err != nil {
					log.Println("id (remote id) value invalid")
					break
				}
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(int64(timeout))*time.Millisecond)
				reply, err := c.Forward(ctx, uint16(remoteId), data)
				if err != nil {
					log.Println("reply", err, string(reply))
				} else {
					log.Println(string(reply))
				}
				cancel()
			default:
				log.Println("invalid type")
			}
		}
		fmt.Print(">>")
	}
}

func initFlagSet() *flag.FlagSet {
	flagSet := flag.NewFlagSet("node-client-cli", flag.ContinueOnError)
	flagSet.Usage = func() {
		flagSet.PrintDefaults()
	}
	flagSet.UintVar(new(uint), "id", 0, "remote id")
	flagSet.UintVar(new(uint), "t", 3000, "timeout/ms")
	flagSet.StringVar(new(string), "type", "request", "send | request")
	return flagSet
}

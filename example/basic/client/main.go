package main

import (
	"flag"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/example/basic"
	"github.com/Li-giegie/node/protocol"
	"log"
	"net"
	"os"
	"time"
)

var (
	localId    = flag.Uint("lid", 1, "local id")
	remoteAddr = flag.String("raddr", "0.0.0.0:8000", "remote addr")
	key        = flag.String("key", "hello", "auth key")
)

func main() {
	flag.Parse()
	client, err := NewClient(uint16(*localId), *key, *remoteAddr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer client.Close()
	log.Printf("client [%d] start success\n", *localId)
	go func() {
		basic.ParseCmd(client, nil)
	}()
	<-client.stopChan
	log.Printf("client [%d] exit\n", *localId)
}

type Client struct {
	common.Conn
	protocol.ClientAuthProtocol
	protocol.ClientHelloProtocol
	stopChan chan error
}

func NewClient(lid uint16, key, rAddr string) (c *Client, err error) {
	c = new(Client)
	c.ClientAuthProtocol = protocol.NewClientAuthProtocol(lid, key, time.Second*3)
	c.ClientHelloProtocol = protocol.NewClientHelloProtocol(time.Second*3, time.Second*12, time.Second*60, nil)
	c.Conn, err = node.Dial("tcp", rAddr, lid, c)
	if err != nil {
		return nil, err
	}
	c.stopChan = make(chan error)
	go c.StartClient(c.Conn)
	return c, nil
}

func (c *Client) Init(conn net.Conn) (remoteId uint16, err error) {
	return c.ClientAuthProtocol.Init(conn)
}

func (c *Client) Connection(conn common.Conn) {
	log.Println("Connection", conn.RemoteId())
}

func (c *Client) Handle(ctx common.Context) {
	log.Printf("client Handle src [%d] %s\n", ctx.SrcId(), ctx.Data())
	ctx.Reply([]byte(fmt.Sprintf("client [%d] handle reply: %s", ctx.DestId(), ctx.Data())))
}

func (c *Client) ErrHandle(msg *common.Message, err error) {
	log.Println("client ErrHandle: ", msg.String(), err)
}

func (c *Client) CustomHandle(ctx common.Context) {
	//log.Println("client CustomHandle: ", ctx.String())
}

func (c *Client) Disconnect(id uint16, err error) {
	log.Println("client Disconnect: ", id, err)
	c.stopChan <- err
}

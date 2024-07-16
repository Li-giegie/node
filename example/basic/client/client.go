package main

import (
	"context"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/protocol"
	"log"
	"net"
	"os"
	"time"
)

type Client struct {
	common.Conn
	protocol.ClientAuthProtocol
	protocol.ClientHelloProtocol
	localId     uint16
	authKey     string
	authTimeout time.Duration
	remoteAddr  string
	stopChan    chan error
}

func (c *Client) Serve() (err error) {
	conn, err := node.Dial("tcp", c.remoteAddr, c.localId, c)
	if err != nil {
		return err
	}
	c.Conn = conn
	go c.ClientHelloProtocol.StartClient(conn)
	return
}

func (c *Client) Init(conn net.Conn) (remoteId uint16, err error) {
	return c.ClientAuthProtocol.Init(conn)
}

func (c *Client) Connection(conn common.Conn) {
	log.Println("Connection", conn.RemoteId())
}

func (c *Client) Handle(ctx common.Context) {
	log.Println("client Handle: ", ctx.String())
	ctx.Reply([]byte("client handle ok"))
}

func (c *Client) ErrHandle(msg *common.Message, err error) {
	log.Println("client ErrHandle: ", msg.String(), err)
}

func (c *Client) CustomHandle(ctx common.Context) {
	if c.ClientHelloProtocol.CustomHandle(ctx) {
		log.Println("client CustomHandle: ", ctx.String())
	}
}

func (c *Client) Disconnect(id uint16, err error) {
	log.Println("client Disconnect: ", id, err)
	c.stopChan <- err
}

func main() {
	client := new(Client)
	client.localId = 1
	client.authTimeout = time.Second * 6
	client.authKey = "hello"
	client.remoteAddr = "127.0.0.1:8080"
	client.ClientAuthProtocol = protocol.NewClientAuthProtocol(client.localId, client.authKey, client.authTimeout)
	client.ClientHelloProtocol = protocol.NewClientHelloProtocol(time.Second, time.Second*3, time.Second*15, os.Stdout)
	client.stopChan = make(chan error, 1)
	err := client.Serve()
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("启动成功")
	log.Println(client.Request(context.Background(), []byte("client request")))
	log.Println(client.Request(context.Background(), []byte("")))
	log.Println(client.Forward(context.Background(), 2, []byte("client forward")))
	log.Println(client.Send([]byte("client send")))
	log.Println(client.WriteMsg(&common.Message{
		Type:   200,
		Id:     0,
		SrcId:  0,
		DestId: client.RemoteId(),
		Data:   []byte("client Custom msg"),
	}))
	time.Sleep(time.Second * 10)
	client.Close()
	<-client.stopChan
}

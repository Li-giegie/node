package main

import (
	"context"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/protocol"
	"log"
	"net"
	"time"
)

type Client struct {
	common.Conn
	localId     uint16
	authKey     string
	authTimeout time.Duration
	addr        string

	stopChan chan error
}

func (c *Client) Serve() error {
	conn, err := node.Dial("tcp", c.addr, c.localId, c)
	if err != nil {
		return err
	}
	c.Conn = conn
	return nil
}

func (c *Client) Connection(conn net.Conn) (remoteId uint16, err error) {
	return protocol.NewAuthProtocol(c.localId, c.authKey, c.authTimeout).ClientNodeHandle(conn)
}

func (c *Client) Handle(ctx common.Context) {
	log.Println("client Handle: ", ctx.String())
	ctx.Reply([]byte("client handle ok"))
}

func (c *Client) ErrHandle(msg *common.Message) {
	log.Println("client ErrHandle: ", msg.String())
}

func (c *Client) DropHandle(msg *common.Message) {
	log.Println("client DropHandle: ", msg.String())
}

func (c *Client) CustomHandle(ctx common.Context) {
	log.Println("client CustomHandle: ", ctx.String())
	ctx.CustomReply(ctx.Type(), []byte("client handle ok"))
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
	client.addr = "127.0.0.1:8080"
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
	client.Close()
	<-client.stopChan
}

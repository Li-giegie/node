package main

import (
	"context"
	"errors"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/utils"
	"log"
	"net"
	"time"
)

type Client struct {
	common.Conn
	id       uint16
	authKey  string
	addr     string
	remoteId uint16
	stopChan chan error
}

type Auth struct {
	ClientId uint16 `json:"client_id,omitempty"`
	ServerId uint16 `json:"server_id,omitempty"`
	Msg      string `json:"msg,omitempty"`
	Permit   bool   `json:"permit,omitempty"`
}

func (c *Client) Serve() error {
	conn, err := node.Dial("tcp", c.addr, c.id, c)
	if err != nil {
		return err
	}
	c.Conn = conn
	return nil
}

func (c *Client) Connection(conn net.Conn) (remoteId uint16, err error) {
	defer log.Println("client connection", remoteId, err)
	auth := new(Auth)
	auth.ClientId = c.id
	auth.Msg = c.authKey
	if err = utils.JSONPackEncode(conn, auth); err != nil {
		return 0, err
	}
	if err = utils.JSONPackDecode(time.Second*6, conn, &auth); err != nil {
		return 0, err
	}
	if !auth.Permit {
		return 0, errors.New("auth fail:" + auth.Msg)
	}
	c.remoteId = auth.ServerId
	return auth.ServerId, nil
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
	client.id = 1
	client.authKey = "hello"
	client.addr = "39.101.193.248:8080"
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
		DestId: client.remoteId,
		Data:   []byte("client Custom msg"),
	}))
	client.Close()
	<-client.stopChan
}

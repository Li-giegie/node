package test

import (
	"context"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/protocol"
	"log"
	"net"
	"testing"
	"time"
)

type ClientHandle struct {
	common.Conn
	*protocol.AuthProtocol
	*protocol.HelloProtocol
	id          uint16
	key         string
	remoteAddr  string
	authTimeout time.Duration
	stopC       chan error
}

func (c *ClientHandle) Init(conn net.Conn) (remoteId uint16, err error) {
	return c.AuthProtocol.ConnectionClient(conn, c.id, c.key, c.authTimeout)
}

func (c *ClientHandle) Connection(conn common.Conn) {

}

func (c *ClientHandle) Handle(ctx common.Context) {
	go func() {
		log.Println("Handle ", ctx.String())
		if err := ctx.Reply([]byte("client 1 handle success")); err != nil {
			fmt.Println(err)
		}
	}()
}

func (c *ClientHandle) ErrHandle(msg *common.Message, err error) {
	log.Println("ErrHandle ", msg.String())
}

func (c *ClientHandle) CustomHandle(ctx common.Context) {
	if c.HelloProtocol.CustomHandle(ctx) {
		log.Println("CustomHandle ", ctx.String())
	}
}

func (c *ClientHandle) Disconnect(id uint16, err error) {
	log.Println("Disconnect ", id, err)
	c.stopC <- err
}

func (c *ClientHandle) Serve() error {
	conn, err := node.Dial("tcp", c.remoteAddr, c.id, c)
	if err != nil {
		return err
	}
	c.AuthProtocol = new(protocol.AuthProtocol)
	c.HelloProtocol = new(protocol.HelloProtocol)
	go c.HelloProtocol.InitClient(conn, time.Second, time.Second*5, time.Second*14, &LogWriter{})
	c.Conn = conn
	return nil
}

func TestClient(t *testing.T) {
	c := &ClientHandle{
		id:          2,
		key:         "hello",
		authTimeout: time.Second * 6,
		remoteAddr:  "0.0.0.0:8080",
		stopC:       make(chan error, 1),
	}
	err := c.Serve()
	if err != nil {
		t.Error(err)
		return
	}
	for i := 0; i < 6; i++ {
		resp, err := c.Request(context.Background(), make([]byte, i))
		if err != nil {
			fmt.Printf("err 第%d次 响应长度%d err长度%d %s\n", i, len(resp), len(err.Error()), err.Error())
		} else {
			fmt.Printf("ok  第%d次 响应长度%d err==nil %v\n", i, len(resp), err == nil)
		}
	}
	fmt.Println(c.Forward(context.Background(), 111, nil))
	////先获取到原始结构体
	//conn := c.Conn.(*common.Connect)
	////创建消息
	//msg := conn.MsgPool.New(conn.LocalId(), 0, 200, []byte("Custom msg"))
	////创建响应接收Chan，把消息Id告诉接收器
	//replyChan := conn.MsgReceiver.Create(msg.Id)
	////发送消息
	//conn.WriteMsg(msg)
	////等待接收
	//replyMsg := <-replyChan
	//log.Println("响应消息", replyMsg.String())
	//c.Close()
	<-c.stopC
}

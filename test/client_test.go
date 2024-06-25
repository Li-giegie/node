package test

import (
	"context"
	"errors"
	"fmt"
	"github.com/Li-giegie/node"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/utils"
	"log"
	"net"
	"sync"
	"testing"
	"time"
)

type ClientHandle struct {
	common.Conn
	stopC chan error
}

type Auth struct {
	ClientId uint16
	Key      string
	ServerId uint16
	Msg      string
	Permit   bool
}

func (c ClientHandle) Connection(conn net.Conn) (remoteId uint16, err error) {
	auth := &Auth{ClientId: 1, Key: "123"}
	if err = utils.JSONPackEncode(conn, auth); err != nil {
		log.Println("auth err 1", err)
		return 0, nil
	}
	if err = utils.JSONPackDecode(time.Second*6, conn, auth); err != nil {
		log.Println("auth err 2", err)
		return 0, err
	}
	if !auth.Permit {
		return 0, errors.New(auth.Msg)
	}
	log.Println("Connection ", conn.RemoteAddr().String())
	return auth.ServerId, nil
}

func (c ClientHandle) Handle(ctx *common.Context) {
	go func() {
		log.Println("Handle ", ctx.String())
		if err := ctx.Reply([]byte("client 1 handle success")); err != nil {
			fmt.Println(err)
		}
	}()
}

func (c ClientHandle) ErrHandle(msg *common.Message) {
	log.Println("ErrHandle ", msg.String())
}

func (c ClientHandle) DropHandle(msg *common.Message) {
	log.Println("DropHandle ", msg.String())
}

func (c ClientHandle) CustomHandle(ctx *common.Context) {
	log.Println("CustomHandle ", ctx.String())
}

func (c ClientHandle) Disconnect(id uint16, err error) {
	log.Println("Disconnect ", id, err)
	c.stopC <- err
}

func TestClient(t *testing.T) {
	c := new(ClientHandle)
	c.stopC = make(chan error, 1)
	conn, err := node.Dial("tcp", "0.0.0.0:8080", 1, c)
	if err != nil {
		t.Error(err)
		return
	}
	c.Conn = conn
	fmt.Println("开始发送")
	wg := sync.WaitGroup{}
	t1 := time.Now()
	//conn.WriteMsg(&common.Message{
	//	Type:   20,
	//	Id:     0,
	//	SrcId:  0,
	//	DestId: 0,
	//	Data:   nil,
	//})
	//conn.Close()
	//return
	//fmt.Println(conn.Forward(context.Background(), 2, []byte("ok")))
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := conn.Request(context.Background(), []byte("client hello 1"))
			if err != nil {
				t.Error(err, data)
				return
			}
			fmt.Println("收到", string(data))
		}()
		fmt.Println(i)
	}
	wg.Wait()
	fmt.Println(time.Since(t1))
	c.Close()
	fmt.Println("结束")
	<-c.stopC
}

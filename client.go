package node

import (
	"context"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/utils"
	"github.com/panjf2000/ants/v2"
	"log"
	"net"
)

type Client interface {
	//HandleFunc 处理方法
	HandleFunc(api uint16, f common.HandleFunc) bool
	//AuthenticationWithServer 向服务端发起认证，认证成功返回连接，ctx：
	AuthenticationWithServer(ctx context.Context, key []byte) (conn common.Conn, err error)
	//SetReceiverPoolSize 接收消息管道复用，减轻gc压力
	SetReceiverPoolSize(n int)
	//SetConstructorPoolSize 发送消息结构体复用，构造器容量，消息复用减轻gc压力，n>0 发送完毕的消息会被放入构造器池等待下一次使用
	SetConstructorPoolSize(n int)
	//SetGoroutinePoolSize 最大开启的协程数
	SetGoroutinePoolSize(n int) (err error)
	//SetMaxReceiveMsgLength 最大接受消息长度，防范内存溢出 n > 0 启用 <=0 不启用
	SetMaxReceiveMsgLength(n int)
}

func NewClient(lid, rid uint16, conn net.Conn) Client {
	c := new(client)
	c.lid = lid
	c.rid = rid
	c.conn = conn
	c.Receiver = common.DEFAULT_Reveiver
	c.ServeMux = common.DEFAULT_ServeMux
	c.Constructor = common.DEFAULT_Constructor
	c.Pool = common.DEFAULT_ClientAntsPool
	return c
}

type client struct {
	lid                 uint16
	rid                 uint16
	maxReceiveMsgLength uint32
	conn                net.Conn
	*common.ServeMux
	*common.Constructor
	*common.Receiver
	*ants.Pool
}

// SetMaxReceiveMsgLength 最大接受消息长度 n > 0 启用 <=0 不启用
func (c *client) SetMaxReceiveMsgLength(n int) {
	if n <= 0 {
		c.maxReceiveMsgLength = 0
	}
	c.maxReceiveMsgLength = uint32(n)
}

func (c *client) SetReceiverPoolSize(n int) {
	c.Receiver = common.NewMessageReceiver(n)
}

func (c *client) SetConstructorPoolSize(n int) {
	c.Constructor = common.NewMessageConstructor(n)
}
func (c *client) SetGoroutinePoolSize(n int) (err error) {
	c.Pool.Release()
	c.Pool, err = ants.NewPool(n)
	return err
}

func (c *client) AuthenticationWithServer(ctx context.Context, key []byte) (conn common.Conn, err error) {
	req := &common.Authenticator{
		SrcId:  c.lid,
		DestId: c.rid,
		Key:    key,
		Type:   common.NodeType_ClientNode,
	}
	defer func() {
		if err != nil {
			_ = c.conn.Close()
		}
	}()
	_, err = c.conn.Write(req.EncodeReq())
	if err != nil {
		return nil, err
	}
	resp := new(common.Authenticator)
	errChan := make(chan error)
	go func() {
		errChan <- resp.CheckErr(resp.DecodeResp(c.conn))
	}()
	select {
	case err = <-errChan:
		if err != nil {
			return
		}
		conn = common.NewConn(c.lid, c.rid, c.conn, c.Constructor, c.Receiver, c, c.maxReceiveMsgLength)
		return
	case <-ctx.Done():
		return nil, &common.ErrTimeout{}
	}
}

func (c *client) Handle(m *common.Message, conn common.Conn) {
	log.Println("warming client handle abnormal error")
}

func DialTCP(remoteId uint16, remoteAddr string, localId uint16, localAddr ...string) (Client, error) {
	if len(localAddr) == 0 {
		localAddr = []string{"0.0.0.0:0"}
	}
	addr := utils.ConvTcpAddr(utils.ParseAddr("tcp", localAddr[0], remoteAddr))
	conn, err := net.DialTCP("tcp", addr[0], addr[1])
	if err != nil {
		return nil, err
	}
	return NewClient(localId, remoteId, conn), nil
}

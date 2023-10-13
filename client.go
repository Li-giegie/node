package node

import (
	"context"
	"errors"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type Client struct {
	id         string
	localAddr  *net.TCPAddr
	remoteAddr *net.TCPAddr
	conn       *ClientConnect
	//连接保活：在一段时间没有发送消息后会发送消息维持连接状态的休眠时间 默认值30s
	KeepAlive time.Duration
	*RouteManager
}

func NewClient(id, remoteAddr string) *Client {
	c := new(Client)
	addr := mustAddress("tcp", remoteAddr, "0.0.0.0:"+getPort())
	c.id = id
	c.remoteAddr = addr[0]
	c.localAddr = addr[1]
	c.KeepAlive = time.Second * 30
	c.RouteManager = newRouter()
	return c
}

func (c *Client) Connect(authenticationData []byte) (*ClientConnect, error) {
	conn, err := net.DialTCP("tcp", c.localAddr, c.remoteAddr)
	if err != nil {
		return nil, err
	}
	err = c.authentication(conn, authenticationData)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	c.conn = newClientConnect(c.id, conn, c.KeepAlive, c.RouteManager)
	return c.conn, nil
}

func (c *Client) authentication(conn *net.TCPConn, authenticationData []byte) error {
	if c.id == "" || strings.Contains(c.id, "\r") {
		return errors.New("authentication err: id Cannot be empty Or contain special characters")
	}
	if err := write(conn, append([]byte(c.id+"\r"), authenticationData...)); err != nil {
		return fmt.Errorf("authentication err: %v", err)
	}
	buf, err := jeans.Unpack(conn)
	if err != nil {
		return fmt.Errorf("authentication err: An exception occurred while receiving data %v", err)
	}
	if len(buf) == 0 {
		return errors.New("authentication err: The content of the reply message is invalid")
	}
	if buf[0] != authenticationSuccess {
		if len(buf) == 1 {
			return errors.New("authentication err: ")
		}
		return errors.New("authentication err: server reply " + string(buf[1:]))
	}
	return nil
}

type ClientConnect struct {
	conn      *net.TCPConn
	response  sync.Map
	activate  int64
	isRunning bool
	localId   string
	r         *RouteManager
}

func newClientConnect(id string, _conn *net.TCPConn, keepAlive time.Duration, r *RouteManager) *ClientConnect {
	conn := new(ClientConnect)
	conn.localId = id
	conn.isRunning = true
	conn.conn = _conn
	conn.activate = time.Now().Unix()
	conn.r = r
	go conn.read()
	go conn.tick(keepAlive)
	return conn
}

func (c *ClientConnect) read() {
	defer c.conn.Close()
	for c.isRunning {
		msg, err := readMessage(c.conn)
		if err != nil {
			log.Printf("client.Conn.read err : at  %v \n", err)
			c.isRunning = false
			return
		}
		c.activate = time.Now().Unix()
		switch msg._type {
		case MsgType_Resp, MsgType_ReqFail, MsgType_ReqForwardFail, MsgType_RespForward, MsgType_RespForwardFail, MsgType_TickResp:
			res, ok := c.response.Load(msg.id)
			if !ok {
				log.Println("收到超时消息或推送消息：", msg.String())
				continue
			}
			res.(chan *Message) <- msg
		case MsgType_ReqForward:
			handle, ok := c.r.api[msg.API]
			ctx := NewContext(msg, c.write)
			if !ok {
				msg._type = MsgType_RespForwardFail
				msg.Data = []byte("remote err: no api")
				msg.localId, msg.remoteId = msg.remoteId, msg.localId
				_ = c.write(msg)
				//_ = ctx.write(ctx.Message)
				continue
			}
			go handle(ctx)
		default:
			log.Println("异常消息", msg.String())
		}

	}

}

func (c *ClientConnect) tick(keepAlive time.Duration) {
	for {
		time.Sleep(keepAlive)
		if time.Now().Unix()-c.activate > int64(keepAlive.Seconds()) {
			ctx, cancel := context.WithTimeout(context.Background(), keepAlive)
			msg, err := c.request(ctx, MsgType_Tick, 0, "", nil)
			log.Println("activate tick：", msg.String())
			if err != nil || msg._type != MsgType_TickResp {
				cancel()
				c.Close()
				log.Fatalln("client tick fail:与服务端断开连接", err)
			}
		}
	}
}

func (c *ClientConnect) write(m *Message) error {
	if !c.isRunning {
		return errors.New("connect is close")
	}
	return write(c.conn, m.Marshal())
}

func (c *ClientConnect) ListenAndServe() {
	for u, _ := range c.r.api {
		log.Printf("[api] %v\n", u)
	}
	log.Println("client listen ------")
	select {}
}

// Request 请求服务端，接受响应
func (c *ClientConnect) Request(ctx context.Context, api uint32, data []byte) (*Message, error) {
	return c.request(ctx, MsgType_Req, api, "", data)
}

func (c *ClientConnect) request(ctx context.Context, _type uint8, api uint32, remoteId string, data []byte) (*Message, error) {
	msg := NewMsg()
	msg._type = _type
	msg.API = api
	msg.remoteId = remoteId
	msg.localId = c.localId
	msg.Data = data
	c.response.Store(msg.id, msg.reply)
	if err := c.write(msg); err != nil {
		close(msg.reply)
		c.response.Delete(msg.id)
		return nil, err
	}
	var reply *Message
	var err error

	select {
	case reply = <-msg.reply:
		if reply._type == MsgType_ReqFail || reply._type == MsgType_ReqForwardFail || reply._type == MsgType_RespForwardFail {
			err = errors.New(string(reply.Data))
		}
	case <-ctx.Done():
		err = errors.New("request err :time out")
	}
	c.response.Delete(msg.id)
	close(msg.reply)
	return reply, err
}

func (c *ClientConnect) RequestForward(ctx context.Context, remoteId string, api uint32, data []byte) (*Message, error) {
	return c.request(ctx, MsgType_ReqForward, api, remoteId, data)
}

func (c *ClientConnect) Close() {

	if !c.isRunning {
		return
	}
	c.isRunning = false
	_ = c.conn.Close()

}

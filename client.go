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
	//检测连接保活的休眠时间 默认值2秒
	DetectionKeepAlive time.Duration
	*RouteManager
}

func NewClient(id, remoteAddr string) *Client {
	c := new(Client)
	addr := mustAddress("tcp", remoteAddr, "0.0.0.0:"+getPort())
	c.id = id
	c.remoteAddr = addr[0]
	c.localAddr = addr[1]
	c.KeepAlive = time.Second * 30
	c.DetectionKeepAlive = time.Second * 2
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
	c.conn = newClientConnect(conn, c.KeepAlive, c.DetectionKeepAlive, c.RouteManager)
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

func (c *Client) Close() {
	if c.conn != nil && c.conn.conn != nil {
		c.conn.isRunning = false
		c.conn.Close()
	}
}

type ClientConnect struct {
	conn      *net.TCPConn
	response  sync.Map
	activate  time.Time
	isRunning bool
	*RouteManager
}

func newClientConnect(_conn *net.TCPConn, keepAlive, detectionKeepAlive time.Duration, r *RouteManager) *ClientConnect {
	conn := new(ClientConnect)
	conn.isRunning = true
	conn.conn = _conn
	conn.activate = time.Now()
	conn.RouteManager = r
	go conn.read()
	go conn.tick(keepAlive, detectionKeepAlive)
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
		switch msg.Type {
		case MessageBaseType_TickReply, MessageBaseType_Response, MessageBaseType_ResponseForward:
			res, ok := c.response.Load(msg.Id)
			if !ok {
				log.Println("收到推送消息：", msg.String())
				continue
			}
			res.(chan *MessageBase) <- msg
		case MessageBaseType_Tick, MessageBaseType_Request, MessageBaseType_RequestForward, MessageBaseType_SingleForward:
			handle, ok := c.RouteManager.api[msg.API]
			ctx := NewContext(c, msg)
			if !ok {
				c.RouteManager.NoApi(ctx)
				continue
			}
			go handle(ctx)
		default:
			log.Println("异常消息", msg.String())
		}

	}

}

func (c *ClientConnect) tick(keepAlive, detectionKeepAlive time.Duration) {
	for {
		time.Sleep(detectionKeepAlive)
		if time.Since(c.activate) > keepAlive {
			ctx, _ := context.WithTimeout(context.Background(), keepAlive)
			msg, err := c.request(ctx, MessageBaseType_Tick, 0, nil)
			if err != nil {
				c.Close()
				log.Fatalln("client tick fail 连接失败", err)
			}
			if len(msg.Data) > 0 && msg.Data[0] == 1 {
				log.Println("client tick success ")
			}
		}
	}
}

func (c *ClientConnect) write(m *MessageBase) error {
	if !c.isRunning {
		return errors.New("connect is close")
	}
	return write(c.conn, m.Marshal())
}

func (c *ClientConnect) send(api uint32, _type uint8, data []byte) (*MessageBase, error) {
	msg := NewMessageBase(api, _type, data)
	if _type == MessageBaseType_Request || _type == MessageBaseType_RequestForward || _type == MessageBaseType_Tick {
		c.response.Store(msg.Id, msg.handleStatus)
	}
	c.activate = time.Now()
	return msg, write(c.conn, msg.Marshal())
}

// 仅发送消息到服务端
func (c *ClientConnect) Send(api uint32, data []byte) error {
	_, err := c.send(api, MessageBaseType_Single, data)

	return err
}

// 请求服务端，接受响应
func (c *ClientConnect) Request(ctx context.Context, api uint32, data []byte) (*MessageBase, error) {
	return c.request(ctx, MessageBaseType_Request, api, data)
}

func (c *ClientConnect) request(ctx context.Context, _type uint8, api uint32, data []byte) (*MessageBase, error) {
	msg, err := c.send(api, _type, data)
	if err != nil {
		return nil, err
	}
	var reply *MessageBase
	select {
	case reply = <-msg.handleStatus:
		break
	case <-ctx.Done():
		err = errors.New("request err :time out")
	}
	c.response.Delete(msg.Id)
	close(msg.handleStatus)
	return reply, err
}

func (c *ClientConnect) SingleForward(srcId, destId string, destApi uint32, data []byte) error {
	buf := NewMessageForward(srcId, destId, data).Marshal()
	msg := newMessageBase(destApi, MessageBaseType_SingleForward, buf)

	return write(c.conn, msg.Marshal())
}

func (c *ClientConnect) RequestForward(ctx context.Context, srcId, destId string, destApi uint32, data []byte) (*MessageBase, error) {
	buf := NewMessageForward(srcId, destId, data).Marshal()
	msg := NewMessageBase(destApi, MessageBaseType_RequestForward, buf)
	c.response.Store(msg.Id, msg.handleStatus)
	err := write(c.conn, msg.Marshal())
	if err != nil {
		c.response.Delete(msg.Id)
		return nil, err
	}
	defer c.response.Delete(msg.Id)
	defer close(msg.handleStatus)

	select {
	case reply := <-msg.handleStatus:
		return reply, nil
	case <-ctx.Done():
		return nil, errors.New("request response timeout")
	}
}

func (c *ClientConnect) Close() {
	c.isRunning = false
	_ = c.conn.Close()
}

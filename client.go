package node

import (
	"context"
	"errors"
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"net"
	"sync"
	"time"
)

type Client struct {
	localAddr  *net.TCPAddr
	remoteAddr *net.TCPAddr
	conn       *ClientConnect
	//连接保活：在一段时间没有发送消息后会发送消息维持连接状态的休眠时间 默认值30s
	KeepAlive time.Duration
	//检测连接保活的休眠时间 默认值2秒
	DetectionKeepAlive time.Duration
}

type ClientConnect struct {
	conn     *net.TCPConn
	response sync.Map
	activate time.Time
}

func (c *ClientConnect) read() {
	defer c.conn.Close()
	for {
		buf, err := jeans.Unpack(c.conn)
		if err != nil {
			log.Fatalf("client.Conn.read err : at jeans.Unpack %v ", err)
		}
		msg, err := NewMessageBaseWithUnmarshal(buf)
		if err != nil {
			log.Printf("client.Conn.read err : at jeans.Unpack %v \n", err)
			continue
		}

		res, ok := c.response.Load(msg.Id)
		if !ok {
			log.Println("收到推送消息：", msg.String())
			continue
		}
		res.(chan *MessageBase) <- msg
		//go func(_res chan *MessageBase, _msg *MessageBase) {
		//	_res <- _msg
		//	c.response.Delete(_msg.Id)
		//}(res.(chan *MessageBase), msg)

	}

}

func (c *ClientConnect) write(api uint32, _type uint8, data []byte) (*MessageBase, error) {
	msg := NewMessageBase(api, _type, data)
	buf, err := msg.Marshal()
	if err != nil {
		return nil, err
	}
	if _type == MessageBaseType_Request || _type == MessageBaseType_RequestTranspond || _type == MessageBaseType_Tick {
		c.response.Store(msg.Id, msg.handleStatus)
	}

	if _, err = c.conn.Write(jeans.Pack(buf)); err != nil {
		return nil, err
	}
	c.activate = time.Now()

	return msg, nil
}

func (c *ClientConnect) Send(api uint32, data []byte) error {
	_, err := c.write(api, MessageBaseType_Single, data)
	return err
}

func (c *ClientConnect) Request(ctx context.Context, api uint32, data []byte) (*MessageBase, error) {
	msg, err := c.write(api, MessageBaseType_Request, data)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, errors.New("request err :time out")
	case reply := <-msg.handleStatus:
		return reply, err
	}
}

func (c *ClientConnect) Close() {
	c.conn.Close()
}

func NewClient(remoteAddr string) (*Client, error) {
	rAddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
	lAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:"+getPort())
	c := new(Client)
	c.localAddr = lAddr
	c.remoteAddr = rAddr
	c.KeepAlive = time.Second * 30
	c.DetectionKeepAlive = time.Second * 2
	return c, err
}

func (c *Client) Connect() (*ClientConnect, error) {
	conn, err := net.DialTCP("tcp", c.localAddr, c.remoteAddr)
	if err != nil {
		return nil, err
	}
	c.conn = &ClientConnect{conn: conn}
	go c.conn.read()
	go c.tick()
	return c.conn, nil
}

func (c *Client) tick() {
	for {
		time.Sleep(c.DetectionKeepAlive)
		if time.Since(c.conn.activate) > c.KeepAlive {
			msg, err := c.conn.write(0, MessageBaseType_Tick, nil)
			if err != nil {
				log.Fatalln("client tick fail 连接失败", err)
			}
			select {
			case reply := <-msg.handleStatus:
				c.conn.activate = time.Now()
				reply.debug()
				if len(reply.GetData()) == 1 && reply.GetData()[0] == 1 {
					log.Println("client tick success")
				} else {
					log.Println("client tick abnormal")
				}
				close(msg.handleStatus)
			case <-time.After(time.Second * 30):
				close(msg.handleStatus)
				log.Fatalln("tick tick fail 超时")
			}
		}
	}
}

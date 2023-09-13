package node

import (
	"context"
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"net"
	"sync"
)

type Client struct {
	localAddr  *net.TCPAddr
	remoteAddr *net.TCPAddr
	conn       *Conn
}

type Conn struct {
	conn *net.TCPConn
	m    map[uint32]chan struct{}
	sm   sync.Map
}

func (c *Conn) read() {
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

		v2, ok := c.sm.Load(msg.Id)
		if !ok {
			log.Println("收到推送消息：", msg.String())
			continue
		}

		go func() { v2.(chan struct{}) <- struct{}{} }()

	}

}

func (c *Conn) write(api uint32, _type uint8, data []byte) (*MessageBase, error) {

	msg := NewMessageBase(api, _type, data)
	buf, err := msg.Marshal()
	if err != nil {
		return nil, err
	}

	c.sm.Store(msg.Id, msg.handleStatus)
	//rwl.RLock()
	//c.m[msg.Id] = msg.handleStatus
	//rwl.RUnlock()

	_, err = c.conn.Write(jeans.Pack(buf))
	return msg, err
}

func (c *Conn) Send(api uint32, data []byte) error {
	_, err := c.write(api, MessageBaseType_Single, data)
	return err
}

var rwl sync.RWMutex

func (c *Conn) Request(ctx context.Context, api uint32, data []byte) (*MessageBase, error) {
	msg, err := c.write(api, MessageBaseType_Request, data)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, err
	case <-msg.handleStatus:
		return msg, err
	}
}

func NewClient(remoteAddr string) (*Client, error) {
	rAddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
	lAddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:"+getPort())
	c := new(Client)
	c.localAddr = lAddr
	c.remoteAddr = rAddr
	return c, err
}

func (c *Client) Connect() (*Conn, error) {
	conn, err := net.DialTCP("tcp", c.localAddr, c.remoteAddr)
	if err != nil {
		return nil, err
	}
	c.conn = &Conn{conn: conn, m: map[uint32]chan struct{}{}}
	go c.conn.read()
	return c.conn, nil
}

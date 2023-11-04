package node

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"net"
	"sync"
	"time"
)

type Client struct {
	id         uint64
	localAddr  string
	remoteAddr string
	conn       *net.TCPConn
	//连接保活：在一段时间没有发送消息后会发送消息维持连接状态的休眠时间 默认值30s
	keepAlive time.Duration
	*Handler
	response sync.Map
	activate int64
	state    bool
}

func NewClient(remoteAddr string, options ...Option) *Client {
	c := new(Client)
	c.id = DEFAULT_ClientID
	c.localAddr = DEFAULT_ClientAddress
	c.keepAlive = DEFAULT_KeepAlive
	c.remoteAddr = remoteAddr
	c.Handler = newRouter()
	c.state = true
	for _, opt := range options {
		opt.(func(*Client) *Client)(c)
	}
	return c
}

func WithClientId(id uint64) Option {
	return func(c *Client) *Client {
		c.id = id
		return c
	}
}

func WithClientLocalIpAddr(addr string) Option {
	return func(c *Client) *Client {
		c.localAddr = addr
		return c
	}
}

func WithClientKeepAlive(t time.Duration) Option {
	return func(c *Client) *Client {
		c.keepAlive = t
		return c
	}
}

func (c *Client) Connect(authenticationData []byte) ([]byte, error) {
	addr, err := parseAddress("tcp", c.localAddr, c.remoteAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTCP("tcp", addr[0], addr[1])
	if err != nil {
		return nil, err
	}
	c.conn = conn
	reply, err := c.authentication(c.conn, authenticationData)
	if err != nil {
		c.CloseImmediately()
		return nil, err
	}
	go c.read()
	go c.tick(c.keepAlive)
	return reply, nil
}

func (c *Client) authentication(conn *net.TCPConn, authenticationData []byte) ([]byte, error) {
	var id_b = make([]byte, 8)
	binary.LittleEndian.PutUint64(id_b, c.id)
	authBuf := append(id_b, authenticationData...)
	if err := write(conn, authBuf); err != nil {
		return nil, fmt.Errorf("%v %v", auth_err_head, err)
	}
	buf, err := jeans.Unpack(conn)
	if err != nil {
		return nil, fmt.Errorf("%v %v", auth_err_head, err)
	}
	if len(buf) == 0 {
		return nil, errors.New(auth_err_head + " reply message is invalid")
	} else if n := bytes.Index(buf, []byte(auth_sucess)); n >= 0 {
		return buf, nil
	} else if n = bytes.Index(buf, []byte(auth_err_head)); n >= 0 {
		return nil, errors.New(string(buf))
	} else {
		panic("Illegal reply")
	}
}

func (c *Client) read() {
	defer c.Close()
	for c.state {
		msg, err := readMessage(c.conn)
		if err != nil {
			if !c.state {
				log.Println("client connect close ------")
				return
			}
			log.Printf("client.Conn.read err : at  %v \n", err)
			c.state = false
			c.Close()
			return
		}
		c.activate = time.Now().Unix()
		switch msg._type {
		case MsgType_RespSuccess, MsgType_RespFail, MsgType_RespForwardSuccess, MsgType_RespForwardFail, MsgType_TickResp:
			res, ok := c.response.Load(msg.id)
			if !ok {
				log.Println("Receive timeout message or push message:", msg.String())
				continue
			}
			res.(chan *message) <- msg
		case MsgType_ReqForward, MsgType_ServerReq, MsgType_Send:
			_handle, ok := c.Handler.handle[msg.API]
			if !ok {
				if msg._type == MsgType_Send {
					continue
				}
				switch msg._type {
				case MsgType_ReqForward:
					msg._type = MsgType_RespForwardFail
				default:
					msg._type = MsgType_ServerRespFail
				}
				msg.Data = []byte(ErrNoApi.Error())
				msg.srcId, msg.dstId = msg.dstId, msg.srcId
				_ = writeMsg(c.conn, msg)
				continue
			}
			go func(m *message, handle HandleFunc) {
				out, err := handle(m.srcId, m.Data)
				switch m._type {
				case MsgType_ReqForward:
					if err != nil {
						msg._type = MsgType_RespForwardFail
						break
					}
					msg._type = MsgType_RespForwardSuccess
				case MsgType_ServerReq:
					if err != nil {
						msg._type = MsgType_ServerRespFail
						break
					}
					msg._type = MsgType_ServerRespSuccess
				case MsgType_Send:
					return
				}
				msg.srcId, msg.dstId = msg.dstId, msg.srcId
				msg.Data = out
				_ = writeMsg(c.conn, msg)
			}(msg, _handle)
		default:
			log.Println("异常消息", msg.String())
		}

	}

}

func (c *Client) tick(keepAlive time.Duration) {
	tick := time.NewTicker(keepAlive)
	for range tick.C {
		if c.activate+int64(keepAlive.Seconds()) <= time.Now().Unix() {
			ctx, cancel := context.WithTimeout(context.Background(), keepAlive)
			m, err := c.request(ctx, newMsgWithTick())
			log.Println("activate tick：", m.String())
			if err != nil || m._type != MsgType_TickResp {
				cancel()
				c.Close()
				m.recycle()
				log.Fatalln("client tick fail:与服务端断开连接", err)
			}
			m.recycle()
			cancel()
		}
	}
}

func (c *Client) Run() {
	for u, _ := range c.handle {
		log.Printf("[api] %v\n", u)
	}
	log.Println("client listen ------")
	select {}
}

// Send 发送到服务端，但不会有响应
func (c *Client) Send(api uint32, data []byte) error {
	m := newMsgWithSend(api, data)
	err := writeMsg(c.conn, m)
	m.recycle()
	return err
}

// Request 请求服务端，接受响应
func (c *Client) Request(ctx context.Context, api uint32, data []byte) ([]byte, error) {
	m := newMsgWithReq(api, data)
	reply, err := c.request(ctx, m)
	defer reply.recycle()
	return reply.Data, err
}

func (c *Client) RequestForward(ctx context.Context, dstId uint64, api uint32, data []byte) ([]byte, error) {
	m := newMsgWithForward(c.id, dstId, api, data)
	reply, err := c.request(ctx, m)
	defer reply.recycle()
	return reply.Data, err
}

func (c *Client) request(ctx context.Context, m *message) (*message, error) {
	if !c.state {
		return nil, errors.New("connect is close")
	}
	replyChan := make(chan *message)
	c.response.Store(m.id, replyChan)
	defer c.response.Delete(m.id)

	err := writeMsg(c.conn, m)
	if err != nil {
		m.recycle()
		return m, err
	}
	m.recycle()

	var respMsg *message
	select {
	case respMsg = <-replyChan:
		switch respMsg._type {
		case MsgType_RespFail, MsgType_RespForwardFail:
			err = errors.New(string(respMsg.Data))
		}
	case <-ctx.Done():
		err = errors.New("err :time out")
	}
	return respMsg, err
}

func (c *Client) Close() {
	if c.conn != nil {
		c.state = false
		if err := c.conn.Close(); err != nil {
			log.Println("close err: ", err)
		}
	}
}

// CloseImmediately 立刻关闭连接
func (c *Client) CloseImmediately() {
	if c.conn != nil {
		c.state = false
		c.conn.SetLinger(0)
		if err := c.conn.Close(); err != nil {
			log.Println("close err: ", err)
		}
		c.conn = nil
	}
}

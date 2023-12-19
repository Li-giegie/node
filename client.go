/*
2023-11-24 13:28:00
增加：
	增加客户端权重属性 weight uint8
	增加客户端接口注册到服务端功能
*/

package node

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"net"
	"sync"
	"time"
)

type ClientI interface {
	Registration(noExposure ...uint32) ([]uint32, error)                                //将handle API注册到服务端，只有error返回值不为nil时，[]uint32返回失败列表
	HandleFunc(api uint32, handle HandleFunc) *Handler                                  //添加处理函数
	HandlerI(ri ...HandlerI) *Handler                                                   //添加多个处理函数接口
	Connect() (authReply []byte, err error)                                             //发起连接：authReply 服务端认证回复 err 是否连接成功
	Run()                                                                               //如果客户端仅作为请求用途此方法效果仅为阻塞主协程，当客户端挂载有handle接口推荐使用
	Send(api uint32, data []byte) error                                                 //向服务端发送一次数据,本次发送是单向的，不会确认消息是否送达，有tcp/ip协议栈保证消息送达
	Request(ctx context.Context, api uint32, data []byte) ([]byte, error)               //发起一个请求，会等待服务端返回数据
	Forward(ctx context.Context, dstId uint64, api uint32, data []byte) ([]byte, error) //转发一个消息到指定客户端上，dstId远端
	Close(fast ...bool)                                                                 //关闭连接 fast 如果为true，连接迅速关闭，tcp/ip不会确认消息是否真的接受完毕，通常在调试程序中使用
}

type Client struct {
	id         uint64
	localAddr  string
	remoteAddr string
	conn       *net.TCPConn
	keepAlive  time.Duration //连接保活：在一段时间没有发送消息后会发送消息维持连接状态的休眠时间 默认值30s
	handler    *Handler
	response   sync.Map
	activate   time.Duration
	state      bool
	authData   []byte
	closeChan  chan struct{}
}

type OptionClient func(*Client) *Client

func NewClient(remoteAddr string, options ...OptionClient) ClientI {
	c := new(Client)
	c.id = DEFAULT_ClientID
	c.localAddr = DEFAULT_ClientAddress
	c.keepAlive = DEFAULT_KeepAlive
	c.remoteAddr = remoteAddr
	c.handler = newHandler()
	c.state = true
	c.closeChan = make(chan struct{})
	for _, opt := range options {
		opt(c)
	}
	return c
}

func WithClientId(id uint64) OptionClient {
	return func(c *Client) *Client {
		c.id = id
		return c
	}
}

func WithClientLocalIpAddr(addr string) OptionClient {
	return func(c *Client) *Client {
		c.localAddr = addr
		return c
	}
}

func WithClientKeepAlive(t time.Duration) OptionClient {
	return func(c *Client) *Client {
		c.keepAlive = t
		return c
	}
}

func WithClientAuthentication(b []byte) OptionClient {
	return func(c *Client) *Client {
		c.authData = b
		return c
	}
}

func (c *Client) HandleFunc(api uint32, handle HandleFunc) *Handler {
	return c.handler.HandleFunc(api, handle)
}

func (c *Client) HandlerI(ri ...HandlerI) *Handler {
	return c.handler.HandlerI(ri...)
}

func (c *Client) Connect() ([]byte, error) {
	addr, err := parseAddress("tcp", c.localAddr, c.remoteAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTCP("tcp", addr[0], addr[1])
	if err != nil {
		return nil, err
	}
	c.conn = conn
	reply, err := c.authentication()
	if err != nil {
		c.Close(true)
		return nil, err
	}
	go c.process()
	go c.tick(c.keepAlive)
	return reply, nil
}

func (c *Client) authentication() ([]byte, error) {
	id, _ := jeans.Encode(c.id)
	authBuf := append(id, c.authData...)
	if err := write(c.conn, authBuf); err != nil {
		return nil, fmt.Errorf("%v %v", auth_err_head, err)
	}
	buf, err := jeans.Unpack(c.conn)
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
		panic("invalid reply")
	}
}

// registration 注册Api：noExposure 不注册API列表,error 不为空时返回注册失败的api
func (c *Client) Registration(noExposure ...uint32) ([]uint32, error) {
	apiList := handleMapToSlice(c.handler.handle, noExposure...)
	msg := newMsgWithRegistration(apiList)
	defer msg.recycle()
	ctx, cancel := context.WithTimeout(context.Background(), DEFAULT_REQUESTTIMEOUT)
	defer cancel()
	msg, err := c.request(ctx, msg)
	var text string
	var badApiListBuf []byte
	if err != nil {
		if err = jeans.Decode(msg.data, &text, &badApiListBuf); err != nil {
			panic("1" + err.Error())
		}
		var badApiList []uint32
		if err = jeans.DecodeSlice(badApiListBuf, &badApiList); err != nil {
			panic("2" + err.Error())
		}
		return badApiList, fmt.Errorf("err : %s", text)
	}
	return nil, nil
}

func (c *Client) process() {
	defer c.Close(true)
	for c.state {
		msg, err := readMessage(c.conn)
		if err != nil {
			if !c.state {
				log.Println("client connect close ------")
				return
			}
			log.Printf("client.Conn.read err : at  %v \n", err)
			c.state = false
			return
		}
		c.activate = time.Duration(time.Now().Unix()) * time.Second
		switch msg.typ {
		case msgType_RespSuccess, msgType_RespFail, msgType_RegistrationSucccess, msgType_RegistrationFail, msgType_ForwardSuccess, msgType_ForwardFail, msgType_TickResp:
			res, ok := c.response.Load(msg.id)
			if !ok {
				log.Println("Receive timeout message or push message:", msg.String(), ok)
				continue
			}
			res.(chan *message) <- msg
		case msgType_Forward, msgType_Req, msgType_Send:
			_handle, ok := c.handler.handle[msg.api]
			if !ok {
				if msg.typ == msgType_Send {
					continue
				}
				switch msg.typ {
				case msgType_Forward:
					msg.typ = msgType_ForwardFail
				default:
					msg.typ = msgType_RespFail
				}
				msg.data = []byte(ErrNoApi.Error())
				msg.srcId, msg.dstId = msg.dstId, msg.srcId
				_ = writeMsg(c.conn, msg)
				continue
			}
			go func(m *message, handle HandleFunc) {
				out, err := handle(m.srcId, m.data)
				switch m.typ {
				case msgType_Forward:
					if err != nil {
						msg.typ = msgType_ForwardFail
						break
					}
					msg.typ = msgType_ForwardSuccess
				case msgType_Req:
					if err != nil {
						msg.typ = msgType_RespFail
						break
					}
					msg.typ = msgType_RespSuccess
				case msgType_Send:
					return
				}
				msg.srcId, msg.dstId = msg.dstId, msg.srcId
				msg.data = out
				_ = writeMsg(c.conn, msg)
			}(msg, _handle)
		default:
			log.Println("异常消息", msg.String())
		}

	}

}

func (c *Client) tick(keepAlive time.Duration) {
	for c.state {
		time.Sleep(time.Second)
		if checkUpTimeOut(c.activate, keepAlive) {
			log.Println("超时")
			ctx, cancel := context.WithTimeout(context.Background(), keepAlive)
			m, err := c.request(ctx, newMsgWithTick())
			if err != nil || m.typ != msgType_TickResp {
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
	for u, _ := range c.handler.handle {
		log.Printf("[api] %v\n", u)
	}
	log.Println("client listen ------")
	<-c.closeChan
}

// Send 发送到服务端，但不会有响应
func (c *Client) Send(api uint32, data []byte) error {
	c.activate = time.Duration(time.Now().Unix()) * time.Second
	return writeMsg(c.conn, newMsgWithSend(api, data))
}

// Request 请求服务端，接受响应
func (c *Client) Request(ctx context.Context, api uint32, data []byte) ([]byte, error) {
	reply, err := c.request(ctx, newMsgWithReq(api, data))
	defer reply.recycle()
	return reply.data, err
}

func (c *Client) Forward(ctx context.Context, dstId uint64, api uint32, data []byte) ([]byte, error) {
	reply, err := c.request(ctx, newMsgWithForward(c.id, dstId, api, data))
	defer reply.recycle()
	return reply.data, err
}

func (c *Client) request(ctx context.Context, m *message) (*message, error) {
	c.activate = time.Duration(time.Now().Unix()) * time.Second
	replyChan := make(chan *message)
	c.response.Store(m.id, replyChan)
	defer c.response.Delete(m.id)
	err := writeMsg(c.conn, m)
	if err != nil {
		return m, err
	}
	replyMsg := msgPool.Get().(*message)
	select {
	case replyMsg = <-replyChan:
		switch replyMsg.typ {
		case msgType_RespFail, msgType_ForwardFail, msgType_RegistrationFail:
			err = errors.New(string(replyMsg.data))
		}
	case <-ctx.Done():
		err = errors.New("err :time out")
	}
	return replyMsg, err
}

// Close 断开连接，可选参数：如果为true：将立即关闭连接不管发送中的数据是否发送完成 避免产生dial tcp x.x.x.x:xxxx->x.x.x.x:xxxx: connectex: Only one usage of each socket address (protocol/network address/port) is normally permitted.
func (c *Client) Close(fast ...bool) {
	if c.closeChan != nil {
		close(c.closeChan)
		c.closeChan = nil
	}
	c.state = false
	if len(fast) > 0 && fast[0] {
		_ = c.conn.SetLinger(0)
	}
	_ = c.conn.Close()
}

/*
2023-11-24 13:28:00
增加：
	增加客户端权重属性 weight uint8
	增加客户端接口注册到服务端功能
*/

package node

import (
	"errors"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	utils "github.com/Li-giegie/go-utils"
	"log"
	"net"
	"time"
)

type ClientI interface {
	Registration(noExposure ...uint32) ([]uint32, error)   //将handle API注册到服务端，只有error返回值不为nil时，[]uint32返回失败列表
	HandleFunc(api uint32, handle HandlerFunc)             //添加处理函数 	//添加多个处理函数接口
	Connect(authData []byte) (authReply []byte, err error) //发起连接：authReply 服务端认证回复 err 是否连接成功
	Run() error                                            //如果客户端仅作为请求用途此方法效果仅为阻塞主协程，当客户端挂载有handle接口推荐使用
	Close(nowait ...bool)
	Send(api uint32, data []byte) error
	Request(timeout time.Duration, api uint32, data []byte) (replyData []byte, err error)
	reply(m *message, typ uint8, data []byte) error
	Forward(timeout time.Duration, dstId uint64, api uint32, data []byte) (replyData []byte, err error)
}

type Client struct {
	id         uint64
	localAddr  string
	remoteAddr string
	keepAlive  time.Duration //连接保活：在一段时间没有发送消息后会发送消息维持连接状态的休眠时间 默认值30s
	handler    *Handler
	msgChan    *utils.MapUint32
	closeChan  chan error
	*connect
}

type OptionClient func(*Client) *Client

func NewClient(remoteAddr string, options ...OptionClient) ClientI {
	c := new(Client)
	c.id = DEFAULT_ClientID
	c.localAddr = DEFAULT_ClientAddress
	c.keepAlive = DEFAULT_KeepAlive
	c.remoteAddr = remoteAddr
	c.handler = NewHandler()
	c.msgChan = utils.NewMapUint32()
	c.closeChan = make(chan error)
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

func (c *Client) HandleFunc(api uint32, handle HandlerFunc) {
	c.handler.Add(api, handle)
}

func (c *Client) Connect(authData []byte) ([]byte, error) {
	addr, err := parseAddress("tcp", c.localAddr, c.remoteAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTCP("tcp", addr[0], addr[1])
	if err != nil {
		return nil, err
	}
	reply, err := c.authentication(conn, authData)
	if err != nil {
		_ = conn.SetLinger(0)
		_ = conn.Close()
		return nil, err
	}
	c.connect = newConnect(c.id, conn, c)
	go c.process()
	go c.tick(c.keepAlive)
	return reply, nil
}

func (c *Client) process() {
	for c.Status {
		tmp, err := readMessage(c.conn)
		if err != nil {
			c.closeChan <- err
			break
		}
		c.activation = time.Now().Unix()
		go func(msg *message) {
			switch msg.typ {
			case msgType_Send:
				ctx := newContext(msg, c)
				hf, ok := c.handler.Get(msg.api)
				if !ok {
					_ = ctx.ReplyErr(errors.New("api not exist"), nil)
					return
				}
				hf(ctx)
			case msgType_Reply, msgType_ReplyErr, msgType_RegistrationResp, msgType_TickResp:
				v, ok := c.msgChan.Get(msg.id)
				if !ok {
					log.Println("receive channel not exit msg drop:", msg.String())
					return
				}
				mChan, ok := v.(chan *message)
				if !ok || mChan == nil {
					log.Println("receive channel not exit or close msg drop:", msg.String())
					return
				}
				mChan <- msg
			default:
				log.Println("drop msg: ", msg.String())
			}
		}(tmp)
	}
}

func (c *Client) authentication(conn *net.TCPConn, data []byte) ([]byte, error) {
	if err := write(conn, encodeAuthReq(c.id, data)); err != nil {
		return nil, fmt.Errorf("%v %v", auth_err_head, err)
	}
	buf, err := jeans.Unpack(conn)
	if err != nil {
		return nil, fmt.Errorf("%v %v", auth_err_head, err)
	}
	return decodeAuthResp(buf)
}

// registration 注册Api：noExposure 不注册API列表,error 不为空时返回注册失败的api
func (c *Client) Registration(noExposure ...uint32) ([]uint32, error) {
	apis := filterApi(c.handler.cache.KeyToSlice(), noExposure)
	data, err := c.request(c.keepAlive, msgType_Registration, c.id, 0, 0, encodeRegistrationApiReq(apis))
	if err != nil {
		return nil, err
	}
	return decodeRegistrationApiResp(data)
}

func (c *Client) tick(keepAlive time.Duration) {
	keepNum := int64(keepAlive.Seconds())
	for c.Status {
		time.Sleep(time.Second)
		if time.Now().Unix() >= c.activation+keepNum {
			log.Println("超时")
			_, err := c.request(c.keepAlive, msgType_Tick, c.id, 0, 0, nil)
			if err != nil {
				log.Println(err)
				c.Close(true)
			}
		}
	}
}

func (c *Client) Run() error {
	c.handler.Range(func(api uint32, ih HandlerFunc) {
		log.Printf("[api] %v\n", api)
	})
	log.Println("client listen ------")
	err := <-c.closeChan
	return err
}

func (c *Client) storageMsgChan(id uint32, mshChan chan *message) {
	c.msgChan.Set(id, mshChan)
}

func (c *Client) Close(nowait ...bool) {
	c.connect.close(nowait...)
}

func (c *Client) Send(api uint32, data []byte) error {
	return c.send(c.id, msgType_Send, api, data)
}

func (c *Client) Request(timeout time.Duration, api uint32, data []byte) (replyData []byte, err error) {
	return c.request(timeout, msgType_Send, c.Id, 0, api, data)
}

func (c *Client) Forward(timeout time.Duration, dstId uint64, api uint32, data []byte) (replyData []byte, err error) {
	return c.request(timeout, msgType_Send, c.Id, dstId, api, data)
}

func (c *Client) reply(m *message, typ uint8, data []byte) error {
	return c.reply(m, typ, data)
}

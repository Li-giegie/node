package node

import (
	"errors"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"net"
	"time"
)

type ClientI interface {
	Registration(noExposure ...uint32) ([]uint32, error)                 //将handle API注册到服务端，只有error返回值不为nil时，[]uint32返回失败列表
	HandleFunc(api uint32, handle HandlerFunc)                           //添加处理函数 	//添加多个处理函数接口
	Connect(dstId uint64, authData []byte) (authReply []byte, err error) //发起连接：入参dstId：目的Id即server id，authData 认证发送的数据，authReply 服务端认证回复 err 是否连接成功
	Run() error                                                          //如果客户端仅作为请求用途此方法效果仅为阻塞主协程，当客户端挂载有handle接口推荐使用
	Close(nowait ...bool)
	Send(api uint32, data []byte) error
	Request(timeout time.Duration, api uint32, data []byte) (replyData []byte, err error)
	Forward(timeout time.Duration, dstId uint64, api uint32, data []byte) (replyData []byte, err error)
}

type Client struct {
	id           uint64
	localAddr    string
	remoteAddr   string
	keepAlive    time.Duration //连接保活：在一段时间没有发送消息后会发送消息维持连接状态的休眠时间 默认值30s
	connDeadline time.Duration
	closeChan    chan error
	activation   int64
	iHandler
	iMessageChan
	*connect
}

type OptionClient func(*Client) *Client

func NewClient(remoteAddr string, options ...OptionClient) ClientI {
	c := new(Client)
	c.id = DEFAULT_ClientID
	c.localAddr = DEFAULT_ClientAddress
	c.keepAlive = DEFAULT_KeepAlive
	c.remoteAddr = remoteAddr
	c.iHandler = newHandler()
	c.iMessageChan = newMessageChan()
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

func (c *Client) getConnDeadline() time.Duration {
	return c.connDeadline
}

func (c *Client) HandleFunc(api uint32, handle HandlerFunc) {
	c.AddHandle(api, handle)
}

func (c *Client) Connect(dstId uint64, authData []byte) ([]byte, error) {
	addr, err := parseAddress("tcp", c.localAddr, c.remoteAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTCP("tcp", addr[0], addr[1])
	if err != nil {
		return nil, err
	}
	reply, err := c.authentication(conn, dstId, authData)
	if err != nil {
		_ = conn.SetLinger(0)
		_ = conn.Close()
		return reply, err
	}
	c.connect = newConnect(c.id, conn, c)
	c.activation = time.Now().Unix()
	go c.process()
	go c.tick(c.keepAlive)
	return reply, nil
}

func (c *Client) process() {
	for c.Status {
		tmp, err := c.read()
		if err != nil {
			c.closeChan <- err
			break
		}
		c.activation = time.Now().Unix()
		go func(msg *message) {
			switch msg.typ {
			case msgType_Send:
				ctx := newContext(msg, c)
				hf, ok := c.GetHandle(msg.api)
				if !ok {
					_ = ctx.ReplyErr(errors.New("api not exist"), nil)
					return
				}
				hf(ctx)
			case msgType_Reply, msgType_ReplyErr, msgType_RegistrationReply, msgType_TickReply:
				mChan, ok := c.GetMsgChan(msg.id)
				if !ok {
					log.Println("receive channel not exit msg drop:", msg.String())
					return
				}
				if mChan == nil {
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
func (c *Client) read() (*message, error) {
	buf, err := readAtLeast(c.conn, msg_headerLen)
	if err != nil {
		return nil, err
	}
	m := msgPool.Get().(*message)
	dl := m.header.unmarshal(buf)
	m.data, err = readAtLeast(c.conn, int(dl))
	return m, err
}
func (c *Client) authentication(conn *net.TCPConn, dst uint64, data []byte) ([]byte, error) {
	buf, err := readAtLeast(conn, 4)
	if err != nil {
		return nil, fmt.Errorf("%v %v", auth_err_head, err)
	}
	sessionId := bytesToUint32(buf)
	if _, err := conn.Write(newAuthMsg(c.id, dst, sessionId, data).marshal()); err != nil {
		return nil, fmt.Errorf("%v %v", auth_err_head, err)
	}
	buf, err = jeans.Unpack(conn)
	if err != nil {
		return nil, fmt.Errorf("%v %v", auth_err_head, err)
	}
	return decodeErrReplyMsgData(buf)
}

// Registration 注册Api：noExposure 不注册API列表,error 不为空时返回注册失败的api
func (c *Client) Registration(noExposure ...uint32) ([]uint32, error) {
	apis := filterApi(c.HandlerKeys(), noExposure)
	msg, err := c.request(c.keepAlive, c.id, 0, msgType_Registration, 0, encodeRegistrationApiReq(apis))
	if err != nil {
		return nil, err
	}
	defer msg.recycle()
	return decodeRegistrationApiResp(msg.data)
}

func (c *Client) tick(keepAlive time.Duration) {
	keepNum := int64(keepAlive.Seconds())
	for c.Status {
		time.Sleep(time.Second)
		if time.Now().Unix() >= c.activation+keepNum {
			_, err := c.request(c.keepAlive, c.id, 0, msgType_Tick, 0, nil)
			if err != nil {
				log.Println(err)
				c.Close(true)
			}
			log.Println("tick------")
		}
	}
}

func (c *Client) Run() error {
	c.RangeHandle(func(api uint32, ih HandlerFunc) {
		log.Printf("[api] %v\n", api)
	})
	log.Printf("client [%d] listen ------\n", c.id)
	err := <-c.closeChan
	return err
}

func (c *Client) Close(nowait ...bool) {
	c.connect.close(nowait...)
}

func (c *Client) Send(api uint32, data []byte) error {
	c.activation = time.Now().Unix()
	return c.send(c.id, 0, msgType_Send, api, data)
}

func (c *Client) Request(timeout time.Duration, api uint32, data []byte) (replyData []byte, err error) {
	msg, err := c.request(timeout, c.id, 0, msgType_Send, api, data)
	if err != nil {
		return nil, err
	}
	defer msg.recycle()
	c.activation = time.Now().Unix()
	return msg.data, nil
}

func (c *Client) Forward(timeout time.Duration, dstId uint64, api uint32, data []byte) (replyData []byte, err error) {
	msg, err := c.request(timeout, c.id, dstId, msgType_Send, api, data)
	if err != nil {
		return nil, err
	}
	defer msg.recycle()
	c.activation = time.Now().Unix()
	return msg.data, nil
}

package node

import (
	"github.com/Li-giegie/node/common"
	"net"
)

type Client interface {
	Serve() (common.Conn, error)
}

type ClientOption func(c *client)

type client struct {
	*common.MsgReceiver
	*common.MsgPool
	common.Connections
	net.Conn
	Handler
	localId uint16
}

func NewClient(conn net.Conn, localId uint16, h Handler, opts ...ClientOption) Client {
	c := new(client)
	c.Conn = conn
	c.localId = localId
	c.Handler = h
	c.MsgPool = common.NewMsgPool(1024)
	c.MsgReceiver = common.NewMsgReceiver(1024)
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *client) Serve() (common.Conn, error) {
	remoteId, err := c.Init(c.Conn)
	if err != nil {
		return nil, err
	}
	nodeConn := common.NewConn(c.localId, remoteId, c.Conn, c, c.Connections, nil)
	go func() {
		nodeConn.Serve(c)
		_ = nodeConn.Close()
	}()
	return nodeConn, nil
}

// Dial network、address，同net.Dail相同，localId 本地节点Id，node 生命周期管理实现接口
func Dial(network, address string, localId uint16, h Handler, opts ...ClientOption) (common.Conn, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewClient(conn, localId, h, opts...).Serve()
}

// Serve 非阻塞启动，net.Conn已建立的连接，localId 本地节点Id，node 生命周期管理实现接口、conns当前节点需要继承另一个服务端节点做边界网关节点时传入服务端节点，如不需要传入nil
func Serve(conn net.Conn, localId uint16, h Handler, opts ...ClientOption) (common.Conn, error) {
	return NewClient(conn, localId, h, opts...).Serve()
}

// WIthClientMsgPoolSize 消息池容量，消息在从池子中创建和销毁，这一行为是考虑到GC压力
func WIthClientMsgPoolSize(n int) ClientOption {
	return func(s *client) {
		s.MsgPool = common.NewMsgPool(n)
		return
	}
}

// WithClientMsgReceivePoolSize 消息接收池容量，消息接收每次创建的Channel从池子中创建和销毁，这一行为是考虑到GC压力
func WithClientMsgReceivePoolSize(n int) ClientOption {
	return func(s *client) {
		s.MsgReceiver = common.NewMsgReceiver(n)
		return
	}
}

func WithClientConnections(conns common.Connections) ClientOption {
	return func(s *client) {
		s.Connections = conns
	}
}

package node

import (
	"context"
	"errors"
	go_jeans "github.com/Li-giegie/go-jeans"
	"github.com/Li-giegie/node/common"
	"github.com/Li-giegie/node/utils"
	"io"
	"net"
	"time"
)

type Client struct {
	*common.MsgReceiver
	lid  uint16
	rid  uint16
	conn net.Conn
}

func NewClient(conn net.Conn, localId, remoteId uint16) *Client {
	c := new(Client)
	c.conn = conn
	c.lid = localId
	c.rid = remoteId
	c.MsgReceiver = common.NewMsgReceiver(1024)
	return c
}

func (c *Client) InitConn(ctx context.Context, h Handler) (Conn, error) {
	connInit := new(ConnInitializer)
	connInit.LocalId = c.lid
	connInit.RemoteId = c.rid
	err := connInit.Send(c.conn)
	if err != nil {
		return nil, err
	}
	if err = connInit.ReceptionWithCtx(ctx, c.conn); err != nil {
		return nil, err
	}
	if err = connInit.Error(); err != nil {
		return nil, err
	}
	conn := common.NewConn(c.lid, c.rid, c.conn, c.MsgReceiver, nil, nil, h)
	go conn.Serve()
	h.Connection(conn)
	return conn, nil
}

// DialTCP 发起tcp连接并启动服务
func DialTCP(ctx context.Context, localId, remoteId uint16, raddr string, h Handler) (Conn, error) {
	conn, err := net.Dial("tcp", raddr)
	if err != nil {
		return nil, err
	}
	return NewClient(conn, localId, remoteId).InitConn(ctx, h)
}

// ConnInitializer 连接初始化，将本地节点Id告知远程节点
type ConnInitializer struct {
	LocalId  uint16
	RemoteId uint16
	code     uint8
	checksum uint32
}

const (
	authCode_UnknownErr uint8 = iota
	authCode_ridErr
	authCode_nodeExist
	authCode_success
)

var nodeExistErr = errors.New("node already exists")
var unknownErr = errors.New("unknown error")
var invalidChecksum = errors.New("invalid checksum")
var remoteIdErr = errors.New("remote id error")

// Error 检查回复是否包含错误错误
func (a *ConnInitializer) Error() error {
	switch a.code {
	case authCode_nodeExist:
		return nodeExistErr
	case authCode_ridErr:
		return remoteIdErr
	case authCode_success:
		return nil
	default:
		return unknownErr
	}
}

// Send 发送初始化信息
func (a *ConnInitializer) Send(w io.Writer) error {
	a.checksum = uint32(byte(a.LocalId) + byte(a.LocalId>>8) + byte(a.RemoteId) + byte(a.RemoteId>>8) + a.code)
	data, _ := go_jeans.EncodeBase(a.LocalId, a.RemoteId, a.code, a.checksum)
	_, err := w.Write(data)
	return err
}

// ReceptionWithCtx ctx 接收远程节点返回的信息
func (a *ConnInitializer) ReceptionWithCtx(ctx context.Context, r io.Reader) (err error) {
	buf := make([]byte, 9)
	if err = utils.ReadFullCtx(ctx, r, buf); err != nil {
		return err
	}
	if err = go_jeans.Decode(buf, &a.LocalId, &a.RemoteId, &a.code, &a.checksum); err != nil {
		return err
	}
	checksum := utils.CheckSum(buf[:5])
	if checksum != a.checksum {
		return invalidChecksum
	}
	return nil
}

// ReceptionWithTimeout 超时接收远程节点返回的信息
func (a *ConnInitializer) ReceptionWithTimeout(t time.Duration, r io.Reader) (err error) {
	buf := make([]byte, 9)
	if err = utils.ReadFull(t, r, buf); err != nil {
		return err
	}
	if err = go_jeans.Decode(buf, &a.LocalId, &a.RemoteId, &a.code, &a.checksum); err != nil {
		return err
	}
	checksum := utils.CheckSum(buf[:5])
	if checksum != a.checksum {
		return invalidChecksum
	}
	return nil
}

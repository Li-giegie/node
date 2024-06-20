package common

import (
	"encoding/binary"
	"errors"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"github.com/Li-giegie/node/utils"
	"io"
	"time"
)

const (
	//心跳消息
	MsgType_Tick uint8 = iota
	MsgType_TickReply

	MsgType_Send
	MsgType_Reply

	MsgType_ReplyErrWithApiNotExist
	MsgType_ReplyErrWithConnectNotExist
	MsgType_ReplyErrWithLenLimit
	MsgType_ReplyErrWithCheckInvalid

	MsgType_ReplyErrWithTimeout
	MsgType_ReplyErrWithWrite
)

const MESSAGE_HEADER_LEN = 4 + 2 + 2 + 2 + 4 + 2

type Message struct {
	Typ        uint8
	Id         uint32
	SrcId      uint16
	DestId     uint16
	Api        uint16
	DataLength uint32
	Data       []byte
}

func (m *Message) String() string {
	return fmt.Sprintf("Message {typ: %v, id: %v, srcId: %v, destId: %v, api: %v,dataLength: %d, data: %s}", m.Typ, m.Id, m.SrcId, m.DestId, m.Api, m.DataLength, m.Data)
}

func (m *Message) Encode() []byte {
	buf := make([]byte, MESSAGE_HEADER_LEN, MESSAGE_HEADER_LEN+len(m.Data))
	buf[0] = m.Typ
	utils.EncodeUint24(buf[1:], m.Id)
	m.DataLength = uint32(len(m.Data))
	binary.LittleEndian.PutUint16(buf[4:], m.SrcId)
	binary.LittleEndian.PutUint16(buf[6:], m.DestId)
	binary.LittleEndian.PutUint16(buf[8:], m.Api)
	binary.LittleEndian.PutUint32(buf[10:], m.DataLength)
	binary.LittleEndian.PutUint16(buf[14:], uint16(m.Typ)^uint16(m.Id)^m.SrcId^m.DestId^m.Api^uint16(m.DataLength))
	return append(buf, m.Data...)
}

func (m *Message) DecodeHeader(r io.Reader, buf []byte) error {
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return err
	}
	m.Typ = buf[0]
	m.Id = utils.DecodeUint824(buf[1:])
	m.SrcId = binary.LittleEndian.Uint16(buf[4:])
	m.DestId = binary.LittleEndian.Uint16(buf[6:])
	m.Api = binary.LittleEndian.Uint16(buf[8:])
	m.DataLength = binary.LittleEndian.Uint32(buf[10:])
	sum := binary.LittleEndian.Uint16(buf[14:])
	if m.Typ != MsgType_Tick && m.Typ != MsgType_TickReply && uint16(m.Typ)^uint16(m.Id)^m.SrcId^m.DestId^m.Api^uint16(m.DataLength) != sum {
		return DEFAULT_ErrMsgCheck
	}
	m.Data = buf[16:]
	return nil
}

func (m *Message) DecodeHeaderWithTimeout(r io.Reader, buf []byte, t time.Duration) error {
	replyC := make(chan error)
	go func() {
		replyC <- m.DecodeHeader(r, buf)
	}()
	select {
	case err := <-replyC:
		return err
	case <-time.After(t):
		return errors.New("timeout")
	}
}

func (m *Message) DecodeContent(r io.Reader) (err error) {
	m.Data, err = utils.ReadAtLeast(r, int(m.DataLength))
	return err
}

func (m *Message) DecodeContentWithTimeout(r io.Reader, t time.Duration) (err error) {
	replyC := make(chan error)
	go func() {
		replyC <- m.DecodeContent(r)
	}()
	select {
	case err = <-replyC:
		return err
	case <-time.After(t):
		return errors.New("timeout")
	}
}
func (m *Message) Reply(typ uint8, data []byte) {
	m.Typ = typ
	m.Data = data
	m.SrcId, m.DestId = m.DestId, m.SrcId
}

const (
	NodeType_ClientNode = iota
	NodeType_ServerNode
)

type Authenticator struct {
	SrcId     uint16
	DestId    uint16
	Type      uint8
	StateCode uint8
	KeyLen    uint32
	Key       []byte
}

func (A *Authenticator) HeaderLen() int {
	return 10
}

func (A *Authenticator) EncodeReq() []byte {
	buf, err := jeans.Encode(A.SrcId, A.DestId, A.Type, A.StateCode, A.Key)
	if err != nil {
		panic(err)
	}
	return buf
}
func (A *Authenticator) DecodeReqHeader(r io.Reader) (err error) {
	buf := make([]byte, A.HeaderLen())
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return err
	}
	return jeans.Decode(buf, &A.SrcId, &A.DestId, &A.Type, &A.StateCode, &A.KeyLen)
}

func (A *Authenticator) DecodeReqHeaderWithTimeout(r io.Reader, t time.Duration) error {
	errChan := make(chan error)
	go func() {
		errChan <- A.DecodeReqHeader(r)
	}()
	select {
	case err := <-errChan:
		return err
	case <-time.After(t):
		return &ErrTimeout{text: "decode Authenticator header"}
	}
}

func (A *Authenticator) DecodeReqContent(r io.Reader) (err error) {
	A.Key = make([]byte, A.KeyLen)
	_, err = io.ReadFull(r, A.Key)
	return err
}

func (A *Authenticator) DecodeReqContentWithTimeout(r io.Reader, t time.Duration) error {
	errChan := make(chan error)
	go func() {
		errChan <- A.DecodeReqContent(r)
	}()
	select {
	case err := <-errChan:
		return err
	case <-time.After(t):
		return &ErrTimeout{text: "decode Authenticator content"}
	}
}

func (A *Authenticator) EncodeResp() []byte {
	return []byte{A.StateCode}
}
func (A *Authenticator) DecodeResp(r io.Reader) (err error) {
	buf := make([]byte, 1)
	_, err = r.Read(buf)
	A.StateCode = buf[0]
	return err
}

func (A *Authenticator) CheckErr(err error) error {
	if err != nil {
		return err
	}
	switch A.StateCode {
	case 0:
		return fmt.Errorf("decode error")
	case 1:
		return fmt.Errorf("authentication key error")
	case 2:
		return fmt.Errorf("authentication destId exist")
	case 3:
		return fmt.Errorf("authentication error")
	case 4:
		return fmt.Errorf("authentication key error")
	case 5:
		return fmt.Errorf("authentication error id exist")
	case 6:
		return fmt.Errorf("authentication server internal error")
	case 7:
		return fmt.Errorf("authentication server reply error")
	case 200:
		return nil
	default:
		return errors.New("invalid auth")
	}
}

func (A *Authenticator) String() string {
	return fmt.Sprintf("Authenticator {SrcId: %v, DestId: %v, Type: %v, StateCode: %v, KeyLen: %d, Key: %s}", A.SrcId, A.DestId, A.Type, A.StateCode, A.KeyLen, A.Key)
}

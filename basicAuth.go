package node

import (
	"errors"
	jeans "github.com/Li-giegie/go-jeans"
	"io"
	"time"
)

type Identity struct {
	Id          uint32
	AuthKey     []byte
	AuthTimeout time.Duration
}

type ClientIdentity struct {
	Id            uint32
	RemoteAuthKey []byte
	Timeout       time.Duration
}

var defaultBasicReq = new(basicAuthReq)
var defaultBasicResp = new(basicAuthResp)

type basicAuthReq struct{}

var errBytesLimit = errors.New("number of bytes exceeds the limit size")

func (basicAuthReq) Send(w io.Writer, id uint32, accessKey []byte) error {
	if len(accessKey) > 65520 {
		return errBytesLimit
	}
	data, _ := jeans.EncodeBase(id, accessKey)
	p, _ := Packet(data)
	_, err := w.Write(p)
	return err
}

func (basicAuthReq) Receive(r io.Reader, t time.Duration) (id uint32, accessKey []byte, err error) {
	data, err := Unpack(r, t)
	if err != nil {
		return 0, nil, err
	}
	err = jeans.DecodeBase(data, &id, &accessKey)
	return
}

type basicAuthResp struct{}

func (basicAuthResp) Send(w io.Writer, id uint32, permit bool, msg string) error {
	if len(msg) > 65520 {
		return errBytesLimit
	}
	data, _ := jeans.EncodeBase(id, permit, msg)
	p, _ := Packet(data)
	_, err := w.Write(p)
	return err
}

func (basicAuthResp) Receive(r io.Reader, t time.Duration) (id uint32, permit bool, msg string, err error) {
	data, err := Unpack(r, t)
	if err != nil {
		return 0, false, "", err
	}
	err = jeans.DecodeBase(data, &id, &permit, &msg)
	return
}

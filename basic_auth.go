package node

import (
	"errors"
	jeans "github.com/Li-giegie/go-jeans"
	"github.com/Li-giegie/node/utils"
	"io"
	"time"
)

type Identity struct {
	Id            uint32
	AccessKey     []byte
	AccessTimeout time.Duration
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
	p, _ := utils.Packet(data)
	_, err := w.Write(p)
	return err
}

func (basicAuthReq) Receive(r io.Reader, t time.Duration) (id uint32, accessKey []byte, err error) {
	data, err := utils.Unpack(r, t)
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
	p, _ := utils.Packet(data)
	_, err := w.Write(p)
	return err
}

func (basicAuthResp) Receive(r io.Reader, t time.Duration) (id uint32, permit bool, msg string, err error) {
	data, err := utils.Unpack(r, t)
	if err != nil {
		return 0, false, "", err
	}
	err = jeans.DecodeBase(data, &id, &permit, &msg)
	return
}

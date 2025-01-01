package internal

import (
	"encoding/binary"
	"github.com/Li-giegie/node/pkg/errors"
	"io"
	"math"
	"time"
)

var DefaultBasicReq = new(BasicAuthReq)
var DefaultBasicResp = new(BasicAuthResp)

type BasicAuthReq struct{}

var maxAuthData uint16 = 40

func (BasicAuthReq) Send(w io.Writer, srcId, dstId uint32, accessKey []byte) error {
	data := make([]byte, 40)
	binary.LittleEndian.PutUint32(data[0:4], srcId)
	binary.LittleEndian.PutUint32(data[4:8], dstId)
	copy(data[8:], Hash(accessKey))
	_, err := w.Write(data)
	return err
}

func (BasicAuthReq) Receive(r io.Reader, t time.Duration) (srcId, dstId uint32, hashKey []byte, err error) {
	var buf = make([]byte, 40)
	err = ReadFull(r, t, buf)
	if err != nil {
		return 0, 0, nil, err
	}
	srcId = binary.LittleEndian.Uint32(buf[0:4])
	dstId = binary.LittleEndian.Uint32(buf[4:8])
	hashKey = buf[8:]
	return
}

type BasicAuthResp struct{}

var maxBasicAuthRespMsgLen = math.MaxUint32 - 100
var errLenOverflow = errors.New("auth response message length overflow")

func (BasicAuthResp) Send(w io.Writer, permit bool, msg string) error {
	if len(msg) > maxBasicAuthRespMsgLen {
		return errLenOverflow
	}
	p := make([]byte, 5+len(msg))
	binary.LittleEndian.PutUint32(p[:4], uint32(len(msg)))
	if permit {
		p[4] = 1
	}
	copy(p[5:], msg)
	_, err := w.Write(p)
	return err
}

func (BasicAuthResp) Receive(r io.Reader, t time.Duration) (permit bool, msg string, err error) {
	buf := make([]byte, 5)
	err = ReadFull(r, t, buf)
	if err != nil {
		return false, "", err
	}
	pl := binary.LittleEndian.Uint32(buf)
	if int(pl) > maxBasicAuthRespMsgLen {
		return false, "", errLenOverflow
	}
	if buf[4] == 1 {
		permit = true
	}
	if pl > 0 {
		buf = make([]byte, pl)
		err = ReadFull(r, t, buf)
		if err != nil {
			return false, "", err
		}
		msg = string(buf)
	}
	return
}

func Auth(rw io.ReadWriter, lid, rid uint32, key []byte, timeout time.Duration) error {
	if err := DefaultBasicReq.Send(rw, lid, rid, key); err != nil {
		return err
	}
	permit, msg, err := DefaultBasicResp.Receive(rw, timeout)
	if err != nil {
		return err
	}
	if !permit {
		return errors.New(msg)
	}
	return nil
}

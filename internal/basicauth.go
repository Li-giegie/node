package internal

import (
	"encoding/binary"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/errors"
	"io"
	"math"
	"time"
)

var DefaultBasicReq = new(BasicAuthReq)
var DefaultBasicResp = new(BasicAuthResp)

type BasicAuthReq struct{}

func (a BasicAuthReq) HeaderLen() int {
	return 41
}

func (a BasicAuthReq) Send(w io.Writer, srcType conn.NodeType, srcId, dstId uint32, accessKey []byte) error {
	data := make([]byte, a.HeaderLen())
	data[0] = byte(srcType)
	binary.LittleEndian.PutUint32(data[1:5], srcId)
	binary.LittleEndian.PutUint32(data[5:9], dstId)
	copy(data[9:], Hash(accessKey))
	_, err := w.Write(data)
	return err
}

func (a BasicAuthReq) Receive(r io.Reader, t time.Duration) (srcType conn.NodeType, srcId, dstId uint32, hashKey []byte, err error) {
	var buf = make([]byte, a.HeaderLen())
	if err = ReadFull(r, t, buf); err != nil {
		return
	}
	srcType = conn.NodeType(buf[0])
	if err = srcType.Valid(); err != nil {
		return
	}
	srcId = binary.LittleEndian.Uint32(buf[1:5])
	dstId = binary.LittleEndian.Uint32(buf[5:9])
	hashKey = buf[9:]
	return
}

type BasicAuthResp struct{}

func (a BasicAuthResp) HeaderLen() int {
	return 6
}

var maxBasicAuthRespMsgLen = math.MaxUint32 - 100
var errLenOverflow = errors.New("auth response message length overflow")

func (a BasicAuthResp) Send(w io.Writer, srcType conn.NodeType, permit bool, msg string) error {
	if len(msg) > maxBasicAuthRespMsgLen {
		return errLenOverflow
	}
	p := make([]byte, a.HeaderLen()+len(msg))
	p[0] = byte(srcType)
	p[1] = bool2uint8(permit)
	binary.LittleEndian.PutUint32(p[2:6], uint32(len(msg)))
	copy(p[6:], msg)
	_, err := w.Write(p)
	return err
}

func (a BasicAuthResp) Receive(r io.Reader, t time.Duration) (dstType conn.NodeType, permit bool, msg string, err error) {
	buf := make([]byte, a.HeaderLen())
	if err = ReadFull(r, t, buf); err != nil {
		return
	}
	dstType = conn.NodeType(buf[0])
	if err = dstType.Valid(); err != nil {
		return
	}
	permit = uint82bool(buf[1])
	pl := binary.LittleEndian.Uint32(buf[2:6])
	if int(pl) > maxBasicAuthRespMsgLen {
		err = errLenOverflow
		return
	}
	if pl > 0 {
		buf = make([]byte, pl)
		if err = ReadFull(r, t, buf); err != nil {
			return
		}
		msg = string(buf)
	}
	return
}

func Auth(rw io.ReadWriter, srcType conn.NodeType, srcId, dstId uint32, key []byte, timeout time.Duration) (dstType conn.NodeType, err error) {
	if err := DefaultBasicReq.Send(rw, srcType, srcId, dstId, key); err != nil {
		return 0, err
	}
	dstType, permit, msg, err := DefaultBasicResp.Receive(rw, timeout)
	if err != nil {
		return 0, err
	}
	if !permit {
		return 0, errors.New(msg)
	}
	return dstType, nil
}

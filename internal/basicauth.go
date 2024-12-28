package internal

import (
	"encoding/binary"
	"errors"
	"io"
	"time"
)

var DefaultBasicReq = new(BasicAuthReq)
var DefaultBasicResp = new(BasicAuthResp)

type BasicAuthReq struct{}

var errBytesLimit = errors.New("number of bytes exceeds the limit size")

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

func (BasicAuthResp) Send(w io.Writer, permit bool, msg string) error {
	if len(msg) > 65530 {
		return errBytesLimit
	}
	p := make([]byte, 3+len(msg))
	binary.LittleEndian.PutUint16(p, uint16(len(msg)))
	if permit {
		p[2] = 1
	}
	copy(p[3:], msg)
	_, err := w.Write(p)
	return err
}

func (BasicAuthResp) Receive(r io.Reader, t time.Duration) (permit bool, msg string, err error) {
	buf := make([]byte, 3)
	err = ReadFull(r, t, buf)
	if err != nil {
		return false, "", err
	}
	pl := binary.LittleEndian.Uint16(buf)
	if pl > 65530 {
		return false, "", errBytesLimit
	}
	if buf[2] == 1 {
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

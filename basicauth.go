package node

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"time"
)

type Identity struct {
	Id      uint32
	Key     []byte
	Timeout time.Duration
}

var defaultBasicReq = new(basicAuthReq)
var defaultBasicResp = new(basicAuthResp)

type basicAuthReq struct{}

var errBytesLimit = errors.New("number of bytes exceeds the limit size")

func (basicAuthReq) Send(w io.Writer, srcId, dstId uint32, accessKey []byte) error {
	data := make([]byte, 40)
	binary.LittleEndian.PutUint32(data[0:4], srcId)
	binary.LittleEndian.PutUint32(data[4:8], dstId)
	copy(data[8:], hash(accessKey))
	_, err := w.Write(data)
	return err
}

func (basicAuthReq) Receive(r io.Reader, t time.Duration) (srcId, dstId uint32, hashKey []byte, err error) {
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

type basicAuthResp struct{}

func (basicAuthResp) Send(w io.Writer, permit bool, msg string) error {
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

func (basicAuthResp) Receive(r io.Reader, t time.Duration) (permit bool, msg string, err error) {
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

func ReadFull(r io.Reader, timeout time.Duration, buf []byte) (err error) {
	errC := make(chan error)
	go func() {
		_, err = io.ReadFull(r, buf)
		errC <- err
	}()
	select {
	case err = <-errC:
		return err
	case <-time.After(timeout):
		return errors.New("timeout")
	}
}

func hash(b []byte) []byte {
	h := sha256.New()
	h.Write(b)
	return h.Sum(nil)
}

package node

import (
	"encoding/binary"
	"errors"
	jeans "github.com/Li-giegie/go-jeans"
	"hash/crc32"
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

type NodeType uint8

const (
	NodeType_Base NodeType = iota
	NodeType_Bridge
)

type basicAuthReq struct{}

var errBytesLimit = errors.New("number of bytes exceeds the limit size")

func (basicAuthReq) Send(w io.Writer, id uint32, accessKey []byte, nt NodeType) error {
	if len(accessKey) > 65520 {
		return errBytesLimit
	}
	data, _ := jeans.EncodeBase(id, accessKey, uint8(nt))
	p, _ := Packet(data)
	_, err := w.Write(p)
	return err
}

func (basicAuthReq) Receive(r io.Reader, t time.Duration) (id uint32, accessKey []byte, nt NodeType, err error) {
	data, err := Unpack(r, t)
	if err != nil {
		return 0, nil, 0, err
	}
	n := uint8(0)
	err = jeans.DecodeBase(data, &id, &accessKey, &n)
	nt = NodeType(n)
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

func ReadFull(timeout time.Duration, r io.Reader, buf []byte) (err error) {
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

// Packet len + len_checksum  + b
func Packet(b []byte) ([]byte, error) {
	l := len(b)
	if l >= 0xfffffff0 {
		return nil, errors.New("b exceeds the length limit")
	}
	buf := make([]byte, 8)
	// 4byte len
	binary.LittleEndian.PutUint32(buf, uint32(l))
	// 4byte len_checksum
	binary.LittleEndian.PutUint32(buf[4:], crc32.ChecksumIEEE(buf[:4]))
	return append(buf, b...), nil
}

func Unpack(r io.Reader, t time.Duration) ([]byte, error) {
	buf := make([]byte, 8)
	err := ReadFull(t, r, buf)
	if err != nil {
		return nil, err
	}
	sum1 := binary.LittleEndian.Uint32(buf[4:])
	sum2 := crc32.ChecksumIEEE(buf[:4])
	if sum1 != sum2 {
		return nil, errors.New("checksum invalid")
	}
	length := binary.LittleEndian.Uint32(buf)
	buf = make([]byte, length)
	err = ReadFull(t, r, buf)
	return buf, err
}

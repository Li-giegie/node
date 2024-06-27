package utils

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"time"
)

var _rnd *rand.Rand

func init() {
	source := rand.NewSource(time.Now().UnixNano())
	_rnd = rand.New(source)
}

var ErrTimeout = errors.New("timeout")

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
		return ErrTimeout
	}
}

func EncodeUint24(b []byte, n2 uint32) {
	n2 = n2 & 0x00FFFFFF
	b[0] = byte(n2)
	b[1] = byte(n2 >> 8)
	b[2] = byte(n2 >> 16)
}

func DecodeUint24(b []byte) (n uint32) {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16
}

func JSONPackEncode(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b, uint32(len(data)))
	binary.LittleEndian.PutUint32(b[4:], uint32(b[0]+b[1]+b[2]+b[3]))
	if _, err = w.Write(b); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

var ErrJsonPackDecode = errors.New("json pack invalid checksum fail")

// JSONPackDecode 从Reader中解码JSON包格式的字符串到结构体中，在经过timeout时间后没有完成结束并返回超时
func JSONPackDecode(timeout time.Duration, r io.Reader, v any) (err error) {
	b := make([]byte, 8)
	if err = ReadFull(timeout, r, b); err != nil {
		return err
	}
	dataLen := binary.LittleEndian.Uint32(b)
	checkSum := binary.LittleEndian.Uint32(b[4:])
	if uint32(b[0]+b[1]+b[2]+b[3]) != checkSum {
		return ErrJsonPackDecode
	}
	data := make([]byte, dataLen)
	if err = ReadFull(timeout, r, data); err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func PackBytes(b []byte) []byte {
	data := make([]byte, 3)
	EncodeUint24(data, uint32(len(b)))
	return append(data, b...)
}

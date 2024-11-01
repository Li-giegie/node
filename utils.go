package node

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"time"
)

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

var ErrLimitPackSize = errors.New("b exceeds the length limit")

// Packet len + len_checksum  + b
func Packet(b []byte) ([]byte, error) {
	l := len(b)
	if l >= 0xfffffff0 {
		return nil, ErrLimitPackSize
	}
	buf := make([]byte, 8)
	// 4byte len
	binary.LittleEndian.PutUint32(buf, uint32(l))
	// 4byte len_checksum
	binary.LittleEndian.PutUint32(buf[4:], crc32.ChecksumIEEE(buf[:4]))
	return append(buf, b...), nil
}

var ErrChecksumInvalid = errors.New("checksum invalid")

func Unpack(r io.Reader, t time.Duration) ([]byte, error) {
	buf := make([]byte, 8)
	err := ReadFull(t, r, buf)
	if err != nil {
		return nil, err
	}
	sum1 := binary.LittleEndian.Uint32(buf[4:])
	sum2 := crc32.ChecksumIEEE(buf[:4])
	if sum1 != sum2 {
		return nil, ErrChecksumInvalid
	}
	length := binary.LittleEndian.Uint32(buf)
	buf = make([]byte, length)
	err = ReadFull(t, r, buf)
	return buf, err
}

func BytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

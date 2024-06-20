package utils

import (
	"encoding/binary"
	"errors"
	"io"
	"math/rand"
	"net"
	"strings"
	"time"
)

var _rnd *rand.Rand

func init() {
	source := rand.NewSource(time.Now().UnixNano())
	_rnd = rand.New(source)
}

func ReadAtLeast(r io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadAtLeast(r, buf, n)
	return buf, err
}

func DeleteRepetition(n []uint32) []uint32 {
	l := len(n)
	newValues := make([]uint32, 0, len(n))
	var ok bool
	for i := 0; i < l; i++ {
		ok = true
		for j := 0; j < len(newValues); j++ {
			if n[i] == newValues[j] {
				ok = false
				break
			}
		}
		if ok {
			newValues = append(newValues, n[i])
		}
	}
	return newValues
}

func RandomU32() uint32 {
	return _rnd.Uint32()<<_rnd.Intn(32) + 1
}

func Uint32ToBytes(n uint32) []byte {
	var b = make([]byte, 4)
	binary.LittleEndian.PutUint32(b, n)
	return b
}

func BytesToUint32(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}

func ParseAddress(protocol string, addr ...string) ([]*net.TCPAddr, error) {
	a := make([]*net.TCPAddr, 0, len(addr))
	for _, item := range addr {
		tmp, err := net.ResolveTCPAddr(protocol, item)
		if err != nil {
			return nil, err
		}
		a = append(a, tmp)
	}
	return a, nil
}

func ConvTcpAddr(addrs []net.Addr) []*net.TCPAddr {
	tcpAddr := make([]*net.TCPAddr, len(addrs))
	for i, addr := range addrs {
		tcpAddr[i] = addr.(*net.TCPAddr)
	}
	return tcpAddr
}

func ParseAddr(network string, addr ...string) []net.Addr {
	a := make([]net.Addr, 0, len(addr))
	switch network {
	case "tcp":
		for _, item := range addr {
			tmp, _ := net.ResolveTCPAddr(network, item)
			a = append(a, tmp)
		}
	case "udp":
		for _, item := range addr {
			tmp, _ := net.ResolveUDPAddr(network, item)
			a = append(a, tmp)
		}
	}
	return a
}

func ReadAtLeastAtBufWithTimeout(r io.Reader, buf []byte, t time.Duration) (err error) {
	errC := make(chan error)
	go func() {
		_, err = ReadAtLeastAtBuf(r, buf)
		errC <- err
	}()
	select {
	case err = <-errC:
		return err
	case <-time.After(t):
		return errors.New("timeout")
	}
}

func ReadAtLeastAtBuf(r io.Reader, buf []byte) (int, error) {
	return io.ReadAtLeast(r, buf, len(buf))
}

func FilterApi(srcApis []uint32, filterApis []uint32) []uint32 {
	var newSrcApis = make([]uint32, 0, len(srcApis))
	var ok bool
	for _, api := range srcApis {
		ok = true
		for _, srcApi := range newSrcApis {
			if api == srcApi {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		for _, u := range filterApis {
			if api == u {
				ok = false
			}
		}
		if ok {
			newSrcApis = append(newSrcApis, api)
		}
	}
	return newSrcApis
}

var portErrStr = []string{
	"bind: Only one usage of each socket address (protocol/network address/port) is normally permitted", //windows
	"bind: address already in use",                //linux
	"bind: Only one usage of each socket address", //windows
}

// IsPortUseErr 判断不通操作系统下net.Dail返回错误是否为端口被占用导致的错误
func IsPortUseErr(err error) bool {
	if err != nil {
		errStr := err.Error()
		for i := 0; i < len(portErrStr); i++ {
			if strings.Contains(errStr, portErrStr[i]) {
				return true
			}
		}
		return false
	}
	return false
}

func BytesEqual(src, dst []byte) bool {
	n := len(src)
	if n != len(dst) {
		return false
	}
	for i := 0; i < n; i++ {
		if src[i] != dst[i] {
			return false
		}
	}
	return true
}

func CountSleep(c bool, n int64, d time.Duration) bool {
	if c {
		d *= time.Duration(n)
		time.Sleep(d)
	}
	return c
}

func EncodeUint24(b []byte, n2 uint32) {
	n2 = n2 & 0x00FFFFFF
	b[0] = byte(n2)
	b[1] = byte(n2 >> 8)
	b[2] = byte(n2 >> 16)
}

func DecodeUint824(b []byte) (n uint32) {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16
}

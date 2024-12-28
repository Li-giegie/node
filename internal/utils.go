package internal

import (
	"crypto/sha256"
	"errors"
	"io"
	"strings"
	"time"
)

func ParseAddr(addr string) (network, address string) {
	if index := strings.Index(addr, "://"); index >= 0 {
		return addr[:index], addr[index+3:]
	}
	return "tcp", addr
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

func Hash(b []byte) []byte {
	h := sha256.New()
	h.Write(b)
	return h.Sum(nil)
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

package node

import (
	"encoding/binary"
	jeans "github.com/Li-giegie/go-jeans"
	"io"
	"math/rand"
	"net"
	"time"
)

var _rnd *rand.Rand

type AuthenticationFunc func(id uint64, data []byte) (reply []byte, err error)

func init() {
	source := rand.NewSource(time.Now().UnixNano())
	_rnd = rand.New(source)
}

func randomU32() uint32 {
	return _rnd.Uint32()<<_rnd.Intn(32) + 1
}

func readAtLeast(r io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadAtLeast(r, buf, n)
	return buf, err
}

// 等待废弃
func write(conn *net.TCPConn, data []byte) error {
	_, err := conn.Write(jeans.Pack(data))
	return err
}

func parseAddress(protocol string, addr ...string) ([]*net.TCPAddr, error) {
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

func filterApi(srcApis []uint32, filterApis []uint32) []uint32 {
	var newSrcApis = make([]uint32, 0, len(srcApis))
	var ok bool
	for _, api := range srcApis {
		ok = true
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

func uint32ToBytes(n uint32) []byte {
	var b = make([]byte, 4)
	binary.LittleEndian.PutUint32(b, n)
	return b
}

func bytesToUint32(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}

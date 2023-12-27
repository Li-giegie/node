package node

import (
	jeans "github.com/Li-giegie/go-jeans"
	"math/rand"
	"net"
	"strconv"
	"time"
)

var _rnd *rand.Rand

type AuthenticationFunc func(id uint64, data []byte) (ok bool, reply []byte)

func init() {
	_rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// 1024-49151
func getPort() string {
	return strconv.Itoa(_rnd.Intn(49152-1024) + 1024)
}

func readMessage(conn *net.TCPConn) (*message, error) {
	buf, err := jeans.Unpack(conn)
	if err != nil {
		return nil, err
	}
	return unmarshalMsg(buf), nil
}

func writeMessage(conn *net.TCPConn, m *message) error {
	buf := m.marshal()
	m.recycle()
	_, err := conn.Write(jeans.Pack(buf))
	return err
}

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

func checkUpTimeOut(t1 time.Duration, to time.Duration) bool {
	return time.Now().Unix() >= int64(t1.Seconds()+to.Seconds())
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

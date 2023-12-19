package node

import (
	"errors"
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
	return newMsgWithUnmarshalV2(buf), nil
}

func writeMsg(conn *net.TCPConn, m *message) error {
	buf := m.marshalV2()
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

type handleRegistrationI interface {
	write(m *message) error
	Id() uint64
	serverConnectionManagerI
}

func serverConnectHandleRegistration(h handleRegistrationI, m *message) ([]uint32, error) {
	var apiList []uint32
	if err := jeans.DecodeSlice(m.data, &apiList); err != nil {
		return nil, h.write(newMsgWithRegistrationResp(m, false, "invalid Registration content", []uint32{}))
	}
	if len(apiList) == 0 {
		return nil, h.write(newMsgWithRegistrationResp(m, false, "invalid Registration not api", []uint32{}))
	}
	var badApiList = make([]uint32, 0, len(apiList))
	var ok bool
	for _, u := range apiList {
		if _, ok = h.GetServerConnectionManager().registrationApi.Get(u); ok {
			badApiList = append(badApiList, u)
			continue
		}
		h.GetServerConnectionManager().registrationApi.Set(u, h.Id())
	}
	var regErr error
	if len(badApiList) == 0 {
		newMsgWithRegistrationResp(m, true, "", nil)
	} else {
		newMsgWithRegistrationResp(m, false, "api exist", badApiList)
		regErr = errors.New("api reg fail")
	}
	if err := h.write(m); err != nil {
		return nil, err
	}
	return apiList, regErr
}

func handleMapToSlice(handler map[uint32]HandleFunc, filter ...uint32) []uint32 {
	res := make([]uint32, 0, len(handler))
	var ok bool
	for u, _ := range handler {
		ok = true
		for _, u2 := range filter {
			if u == u2 {
				ok = false
				break
			}
		}
		if ok {
			res = append(res, u)
		}
	}
	return res
}

func checkUpTimeOut(t1 time.Duration, to time.Duration) bool {
	return time.Now().Unix() >= int64(t1.Seconds()+to.Seconds())
}

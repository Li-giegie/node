package node_register

import (
	"errors"
	jeans "github.com/Li-giegie/go-jeans"
	"strconv"
)

const (
	version = 255
)

type message struct {
	version  uint8
	weight   uint16
	id       uint64
	apiList  []uint32
	authData []byte
}

func (m *message) marshal() []byte {
	buf, err := jeans.EncodeSlice(m.apiList)
	if err != nil {
		panic(err)
	}
	buf, err = jeans.Encode(m.version, m.weight, m.id, buf, m.authData)
	if err != nil {
		panic(err)
	}
	return buf
}

func (m *message) unmarshal(b []byte) error {
	var apiBuf []byte
	err := jeans.Decode(b, &m.version, &m.weight, &m.id, &apiBuf, &m.authData)
	if err != nil {
		panic(err)
	}
	if m.version != version {
		return errors.New("version invalid：" + strconv.Itoa(int(m.version)))
	}
	if err = jeans.DecodeSlice(apiBuf, &m.apiList); err != nil {
		panic(err)
	}
	return nil
}

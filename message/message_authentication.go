package message

import (
	"errors"
	jeans "github.com/Li-giegie/go-jeans"
	"github.com/Li-giegie/node"
)

var ErrChecksumInvalid = errors.New("checksum invalid")

type authReq struct {
	version  uint8
	checksum uint32
	data     []byte
	IMessage
}

func newAuthReq(data []byte) *authReq {
	m := new(authReq)
	m.data = data
	m.version = Version
	return m
}

func (m *authReq) init(im IMessage) {
	m.IMessage = im
}

func (m *authReq) typ() uint8 {
	return message_typ_auth
}

func (m *authReq) checkSum(isUpdate bool) (uint32, error) {
	n, err := checksums(m.version, m.SrcId(), m.DstId(), message_typ_auth, m.data)
	if err != nil {
		return 0, err
	}
	if isUpdate {
		m.checksum = n
	}
	return n, nil
}

// marshalAuthReq field: 1.version 2.checksum 3.dataLength 4.data
func (m *authReq) marshal() ([]byte, error) {
	if _, err := m.checkSum(true); err != nil {
		return nil, err
	}
	return jeans.Encode(m.version, m.checksum, m.data)
}

// unmarshal
func (m *authReq) unmarshal(buf []byte) error {
	err := jeans.Decode(buf, &m.version, &m.checksum, &m.data)
	if err != nil {
		return err
	}
	if n := len(m.data); n > 0 {
		n, err := m.checkSum(false)
		if err != nil {
			return err
		}
		if n != m.checksum {
			return ErrChecksumInvalid
		}
	}
	return nil
}

type authResp struct {
	badApis []uint32
	err     error
}

func (m *message) marshalAuthResp() ([]byte, error) {

	return nil, nil
}

func (m *message) unmarshalAuthResp() error {

	return nil
}

func checksums(args ...interface{}) (uint32, error) {
	buf, err := jeans.Encode(args...)
	if err != nil {
		return 0, err
	}
	return node.Checksum(buf), nil
}

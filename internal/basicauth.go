package internal

import (
	"encoding/binary"
	"github.com/Li-giegie/node/pkg/conn"
	"github.com/Li-giegie/node/pkg/errors"
	"io"
	"time"
)

var DefaultAuthService = new(BaseAuthService)

type BaseAuthRequest struct {
	ConnType conn.NodeType
	SrcId    uint32
	DstId    uint32
	Key      []byte
}

func (r *BaseAuthRequest) Len() int {
	return 41
}

func (r *BaseAuthRequest) Encode() []byte {
	buf := make([]byte, r.Len())
	buf[0] = byte(r.ConnType)
	binary.LittleEndian.PutUint32(buf[1:5], r.SrcId)
	binary.LittleEndian.PutUint32(buf[5:9], r.DstId)
	copy(buf[9:], Hash(r.Key))
	return buf
}

func (r *BaseAuthRequest) Decode(buf []byte) (err error) {
	if len(buf) != r.Len() {
		return errors.New("decode bad: request length invalid")
	}
	r.ConnType = conn.NodeType(buf[0])
	if err = r.ConnType.Valid(); err != nil {
		return err
	}
	r.SrcId = binary.LittleEndian.Uint32(buf[1:5])
	r.DstId = binary.LittleEndian.Uint32(buf[5:9])
	r.Key = buf[9:]
	return nil
}

type BaseAuthService struct {
}

func (s *BaseAuthService) Request(w io.Writer, req *BaseAuthRequest) (err error) {
	_, err = w.Write(req.Encode())
	return err
}

func (s *BaseAuthService) ReadRequest(r io.Reader, timeout time.Duration) (req *BaseAuthRequest, err error) {
	req = new(BaseAuthRequest)
	buf := make([]byte, req.Len())
	if err = ReadFull(r, timeout, buf); err != nil {
		return nil, err
	}
	if err = req.Decode(buf); err != nil {
		return nil, err
	}
	return req, req.ConnType.Valid()
}

type BaseAuthResponseCode uint8

const (
	BaseAuthResponseCodeInvalidSrcId BaseAuthResponseCode = iota + 1
	BaseAuthResponseCodeInvalidDestId
	BaseAuthResponseCodeInvalidKey
	BaseAuthResponseCodeSrcIdExists
	BaseAuthResponseCodeSuccess
)

func (b BaseAuthResponseCode) String() string {
	switch b {
	case BaseAuthResponseCodeInvalidSrcId:
		return "invalid src"
	case BaseAuthResponseCodeInvalidDestId:
		return "invalid dest"
	case BaseAuthResponseCodeInvalidKey:
		return "invalid key"
	case BaseAuthResponseCodeSrcIdExists:
		return "src id exists"
	case BaseAuthResponseCodeSuccess:
		return "success"
	default:
		return "invalid code"
	}
}

func (b BaseAuthResponseCode) Valid() error {
	if b >= BaseAuthResponseCodeInvalidSrcId && b <= BaseAuthResponseCodeSuccess {
		return nil
	}
	return errors.New("invalid code")
}

type BaseAuthResponse struct {
	ConnType              conn.NodeType
	Code                  BaseAuthResponseCode
	MaxMsgLen             uint32
	KeepaliveTimeout      time.Duration
	KeepaliveTimeoutClose time.Duration
}

func (r *BaseAuthResponse) Len() int {
	return 22
}

func (r *BaseAuthResponse) Encode() []byte {
	buf := make([]byte, r.Len())
	buf[0] = byte(r.ConnType)
	buf[1] = byte(r.Code)
	binary.LittleEndian.PutUint32(buf[2:6], r.MaxMsgLen)
	binary.LittleEndian.PutUint64(buf[6:14], uint64(r.KeepaliveTimeout))
	binary.LittleEndian.PutUint64(buf[14:], uint64(r.KeepaliveTimeoutClose))
	return buf
}

func (r *BaseAuthResponse) Decode(buf []byte) (err error) {
	if len(buf) != 22 {
		return errors.New("decode bad: response length invalid")
	}
	r.ConnType = conn.NodeType(buf[0])
	if err = r.ConnType.Valid(); err != nil {
		return err
	}
	r.Code = BaseAuthResponseCode(buf[1])
	if err = r.Code.Valid(); err != nil {
		return err
	}
	r.MaxMsgLen = binary.LittleEndian.Uint32(buf[2:6])
	r.KeepaliveTimeout = time.Duration(binary.LittleEndian.Uint64(buf[6:14]))
	r.KeepaliveTimeoutClose = time.Duration(binary.LittleEndian.Uint64(buf[14:]))
	return nil
}

func (s *BaseAuthService) Response(w io.Writer, resp *BaseAuthResponse) (err error) {
	_, err = w.Write(resp.Encode())
	return
}

func (s *BaseAuthService) ReadResponse(r io.Reader, timeout time.Duration) (resp *BaseAuthResponse, err error) {
	resp = new(BaseAuthResponse)
	buf := make([]byte, resp.Len())
	err = ReadFull(r, timeout, buf)
	if err != nil {
		return nil, err
	}
	if err = resp.Decode(buf); err != nil {
		return nil, err
	}
	return resp, nil
}

package protocol

import (
	"errors"
	"github.com/Li-giegie/node/utils"
	"net"
	"time"
)

type AuthProtocol struct {
	Id      uint16
	Key     string
	Timeout time.Duration
}

func NewAuthProtocol(localId uint16, key string, duration time.Duration) *AuthProtocol {
	return &AuthProtocol{
		Id:      localId,
		Key:     key,
		Timeout: duration,
	}
}

type Auth struct {
	Id     uint16 `json:"id,omitempty"`
	Key    string `json:"key,omitempty"`
	Permit bool   `json:"permit,omitempty"`
	Msg    string `json:"msg,omitempty"`
}

func (a *AuthProtocol) ServerNodeHandle(conn net.Conn) (remoteId uint16, err error) {
	auth := new(Auth)
	defer func() {
		result := new(Auth)
		if err != nil {
			result.Permit = false
			result.Msg = err.Error()
		} else {
			remoteId = auth.Id
			result.Id = a.Id
			result.Permit = true
			result.Msg = "success"
		}
		if err2 := utils.JSONPackEncode(conn, result); err2 != nil {
			err = err2
		}
		if err != nil {
			_ = conn.Close()
		}
	}()
	if err = utils.JSONPackDecode(a.Timeout, conn, auth); err != nil {
		return 0, err
	}
	if auth.Key != a.Key || auth.Id == a.Id {
		return 0, errors.New("key invalid or id clash")
	}
	return
}

func (a *AuthProtocol) ClientNodeHandle(conn net.Conn) (remoteId uint16, err error) {
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	if err = utils.JSONPackEncode(conn, &Auth{Id: a.Id, Key: a.Key}); err != nil {
		return 0, err
	}
	result := new(Auth)
	if err = utils.JSONPackDecode(a.Timeout, conn, result); err != nil {
		return 0, err
	}
	if !result.Permit {
		return 0, errors.New(result.Msg)
	}
	return result.Id, nil
}

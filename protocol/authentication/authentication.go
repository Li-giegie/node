package authentication

import (
	"errors"
	"fmt"
	"github.com/Li-giegie/node/utils"
	"net"
	"time"
)

type AuthProtocol struct{}

type ProtoMsg struct {
	Id     uint16 `json:"id,omitempty"`
	Key    string `json:"key,omitempty"`
	Permit bool   `json:"permit,omitempty"`
	Msg    string `json:"msg,omitempty"`
}

func (a *AuthProtocol) InitServer(conn net.Conn, id uint16, key string, timeout time.Duration) (remoteId uint16, err error) {
	auth := new(ProtoMsg)
	defer func() {
		result := new(ProtoMsg)
		if err != nil {
			result.Permit = false
			result.Msg = err.Error()
		} else {
			remoteId = auth.Id
			result.Id = id
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
	if err = utils.JSONPackDecode(timeout, conn, auth); err != nil {
		return 0, err
	}
	if auth.Key != key || auth.Id == id {
		fmt.Println(auth.Key, auth.Id, id)
		return 0, errors.New("key invalid or id clash")
	}
	return
}

func (a *AuthProtocol) InitClient(conn net.Conn, localId uint16, key string, timeout time.Duration) (remoteId uint16, err error) {
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	if err = utils.JSONPackEncode(conn, &ProtoMsg{Id: localId, Key: key}); err != nil {
		return 0, err
	}
	result := new(ProtoMsg)
	if err = utils.JSONPackDecode(timeout, conn, result); err != nil {
		return 0, err
	}
	if !result.Permit {
		return 0, errors.New(result.Msg)
	}
	return result.Id, nil
}

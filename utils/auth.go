package utils

import (
	"errors"
	"net"
	"time"
)

type AuthInfo struct {
	Id     uint16 `json:"id,omitempty"`
	Key    string `json:"key,omitempty"`
	Permit bool   `json:"permit,omitempty"`
	Msg    string `json:"msg,omitempty"`
}

// AuthHandle 处理认证，在服务端节点中使用
func AuthHandle(conn net.Conn, lid uint16, lKey string, timeout time.Duration) (remoteId uint16, err error) {
	auth := new(AuthInfo)
	defer func() {
		result := new(AuthInfo)
		if err != nil {
			result.Permit = false
			result.Msg = err.Error()
		} else {
			remoteId = auth.Id
			result.Id = lid
			result.Permit = true
			result.Msg = "success"
		}
		if err2 := JSONPackEncode(conn, result); err2 != nil {
			err = err2
		}
		if err != nil {
			_ = conn.Close()
		}
	}()
	if err = JSONPackDecode(timeout, conn, auth); err != nil {
		return 0, err
	}
	if auth.Key != lKey || auth.Id == lid {
		return 0, errors.New("key invalid or id clash")
	}
	return
}

// Auth 发起认证，在客户端节点中使用
func Auth(conn net.Conn, lid uint16, rKey string, timeout time.Duration) (remoteId uint16, err error) {
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	if err = JSONPackEncode(conn, &AuthInfo{Id: lid, Key: rKey}); err != nil {
		return 0, err
	}
	result := new(AuthInfo)
	if err = JSONPackDecode(timeout, conn, result); err != nil {
		return 0, err
	}
	if !result.Permit {
		return 0, errors.New(result.Msg)
	}
	return result.Id, nil
}

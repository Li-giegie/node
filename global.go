package node

import (
	"errors"
	"time"
)

const (
	DEFAULT_ServerAddress         = "0.0.0.0:8088"
	DEFAULT_ClientAddress         = "0.0.0.0:20239"
	DEFAULT_ClientID              = 20230
	DEFAULT_ServerID              = 20240
	DEFAULT_MAXCONNNUM            = 10 * 10000
	DEFAULT_MAX_GOROUTINE         = 10000
	DEFAULT_MIN_GOROUTINE         = 5000
	DEFAULT_KeepAlive             = time.Second * 30
	DEFAULT_AuthenticationTimeout = time.Second * 6
	DEFAULT_CheckInterval         = time.Second * 3
)

var auth_sucess = "authentication success:"
var auth_err_head = "authentication fail:"
var auth_err_conn_supper_limit = errors.New(auth_err_head + " number of connections established by the server reached the upper limit and the connection was denied")
var auth_err_illegality = errors.New(auth_err_head + " Illegal connection")
var auth_err_illegalityIdIsNull = errors.New(auth_err_head + " id is null")
var auth_err_user_online = errors.New(auth_err_head + " User id exist or online")
var auth_err_id_invalid = errors.New(auth_err_head + " User id Cannot be 0")

var (
	ErrConnNotExist         = errors.New("err: id not exist or offline")
	ErrNoApi                = errors.New("err: api not exist")
	ErrDisconnect           = errors.New("err: disconnect")
	ErrInvalid              = errors.New("err: invalid request or send")
	ErrRegistrationApiExist = errors.New("err: Registration api exist")
	ErrTimeout              = errors.New("timeout")
)

// server error list
var (
	ErrServerConnectOverFlow  = errors.New("server busy Please try again later")
	ErrReadConnectErr         = errors.New("read buf error")
	ErrAuthIdEqual0OrServerId = errors.New("auth id equal 0 or equal server id")
	ErrAuthIdExist            = errors.New("auth id exist")
	ErrInvalidConnect         = errors.New("invalid connect")
)

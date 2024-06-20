package common

import (
	"github.com/panjf2000/ants/v2"
	"time"
)

const (
	DEFAULT_Max_Conn_Size         = 1000
	DEFAULT_ServerAntsPoolSize    = 50000
	DEFAULT_ClientAntsPoolSize    = 10000
	DEFAULT_AuthenticationTimeout = time.Second * 6
)

var DEFAULT_ServeMux = NewServeMux()
var DEFAULT_Constructor = NewMessageConstructor(1024)
var DEFAULT_Reveiver = NewMessageReceiver(1024)
var DEFAULT_ServerAntsPool, _ = ants.NewPool(DEFAULT_ServerAntsPoolSize)
var DEFAULT_ClientAntsPool, _ = ants.NewPool(DEFAULT_ClientAntsPoolSize)
var DEFAULT_Conns = NewConns()
var DEFAULT_MaxReceiveMsgLength uint32 = 8192
var DEFAULT_ErrMsgLenLimit = new(ErrMsgLenLimit)
var DEFAULT_ErrMsgCheck = new(ErrMsgCheck)

type ErrMsgLenLimit struct {
}

func (*ErrMsgLenLimit) Error() string {
	return "message length exceeds the limit size"
}

type ErrMsgCheck struct {
}

func (*ErrMsgCheck) Error() string {
	return "message header invalid check"
}

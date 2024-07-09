package protocol

import "github.com/Li-giegie/node/common"

type Conns interface {
	GetConns() []common.Conn
}

var defaultMsgType = common.Null

func GetMsgType() uint8 {
	defaultMsgType++
	return defaultMsgType
}

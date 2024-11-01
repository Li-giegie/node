package protocol

import "github.com/Li-giegie/node/net"

var defaultMsgType = net.Null

func GetMsgType() uint8 {
	defaultMsgType++
	return defaultMsgType
}

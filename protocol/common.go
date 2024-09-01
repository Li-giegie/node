package protocol

import "github.com/Li-giegie/node/common"

var defaultMsgType = common.Null

func GetMsgType() uint8 {
	defaultMsgType++
	return defaultMsgType
}

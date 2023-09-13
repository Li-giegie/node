package node

import (
	jeans "github.com/Li-giegie/go-jeans"
	"log"
	"time"
)

const (
	DEFAULT_messageBaseIdStep uint32 = 1
	DEFAULT_ServerAddress            = "127.0.0.1:2023"
	DEFAULT_ClientAddress            = "127.0.0.1:20239"
)

const (
	//心跳消息
	MessageBaseType_Tick uint8 = iota

	//单程消息：不需要回复的消息
	MessageBaseType_Single

	//请求消息：需要回复的消息
	MessageBaseType_Request

	//单程转发消息：转发后需要回复的消息
	MessageBaseType_SingleTranspond

	//转发请求消息：转发后需要回复的消息
	MessageBaseType_RequestTranspond
)

var MessageBaseTypeMap = map[uint8]string{
	MessageBaseType_Single:           "Single",
	MessageBaseType_Request:          "Request",
	MessageBaseType_SingleTranspond:  "SingleTranspond",
	MessageBaseType_RequestTranspond: "RequestTranspond",
}

func defaultTickHandle() HandlerFunc {
	return func(ctx *Context) {
		log.Println("NoRouteHandle Action ", ctx.String())
	}
}

func defaultNoRouteHandle() HandlerFunc {
	return func(ctx *Context) {
		log.Println("TickHandle Action ", ctx.String())
		buf, _ := jeans.BaseTypeToBytes(time.Now().UnixNano())
		if err := ctx.Reply(buf); err != nil {
			log.Println("TickHandle reply err:", err)
		}
	}
}

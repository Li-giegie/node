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
	DEFAULT_ServerKey                = "1qWa5Xz>l7:P/{].)IhjGbG'"
	DEFAULT_ClientID                 = "node-client"
	DEFAULT_ServerID                 = "node-server"
	authenticationSuccess     byte   = 49
)

const (
	//心跳消息
	MessageBaseType_Tick uint8 = iota
	MessageBaseType_TickReply

	//单程消息：不需要回复的消息
	MessageBaseType_Single

	//请求消息：需要回复的消息
	MessageBaseType_Request
	MessageBaseType_Response

	//单程转发消息：转发后需要回复的消息
	MessageBaseType_SingleForward

	//转发请求消息：转发后需要回复的消息
	MessageBaseType_RequestForward
	MessageBaseType_ResponseForward
)

var MessageBaseTypeMap = map[uint8]string{
	MessageBaseType_Single:          "Single",
	MessageBaseType_Request:         "Request",
	MessageBaseType_Response:        "Response",
	MessageBaseType_SingleForward:   "SingleTranspond",
	MessageBaseType_RequestForward:  "RequestTranspond",
	MessageBaseType_ResponseForward: "ResponseForward",
	MessageBaseType_Tick:            "Tick",
	MessageBaseType_TickReply:       "TickReply",
}

func defaultTickHandle() HandlerFunc {
	return func(ctx *Context) {
		log.Println("TickHandle activate ", ctx.String())
		err := ctx.Write([]byte{1})
		if err != nil {
			log.Println("TickHandle activate reply fail ", ctx.String())
			return
		}
	}
}

func defaultNoRouteHandle() HandlerFunc {
	return func(ctx *Context) {
		log.Println("NoRouteHandle Action ", ctx.String())
		buf, _ := jeans.BaseTypeToBytes(time.Now().UnixNano())
		if err := ctx.Write(buf); err != nil {
			log.Println("NoRouteHandle reply err:", err)
		}
	}
}

func defaultAbnormalApiHandle() HandlerFunc {
	return func(ctx *Context) {
		log.Println("AbnormalApiHandle Action ", ctx.String())
		ctx.Close()
	}
}

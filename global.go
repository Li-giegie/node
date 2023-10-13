package node

const (
	DEFAULT_ServerAddress      = "127.0.0.1:2023"
	DEFAULT_ClientAddress      = "127.0.0.1:20239"
	DEFAULT_ServerKey          = "1qWa5Xz>l7:P/{].)IhjGbG'"
	DEFAULT_ClientID           = "node-client"
	DEFAULT_ServerID           = "node-server"
	authenticationSuccess byte = 49
	DEFAULT_MAXCONNNUM         = 10 * 10000
)

const (
	//心跳消息
	MsgType_Tick uint8 = iota
	MsgType_TickResp

	//请求消息：需要回复的消息
	MsgType_Req
	MsgType_ReqFail
	MsgType_Resp

	//转发请求消息：转发后需要回复的消息
	MsgType_ReqForward
	MsgType_ReqForwardFail

	MsgType_RespForward
	MsgType_RespForwardFail
)

var MsgTypeMap = map[uint8]string{
	MsgType_Req:     "MsgType_Req",
	MsgType_Resp:    "MsgType_Resp",
	MsgType_ReqFail: "MsgType_ReqFail",

	MsgType_ReqForward: "MsgType_ReqForward",

	MsgType_ReqForwardFail:  "MsgType_ReqForwardFail",
	MsgType_RespForward:     "MsgType_RespForward",
	MsgType_RespForwardFail: "MsgType_RespForwardFail",

	MsgType_Tick:     "Tick",
	MsgType_TickResp: "TickRespOk",
}

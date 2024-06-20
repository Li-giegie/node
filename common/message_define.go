package common

import "fmt"

type IMsgErrType_ConnectNotExist_Error interface {
	Error() string
	Type() uint8
}

type MsgErrType_ConnectNotExist_Error struct {
}

func (MsgErrType_ConnectNotExist_Error) Error() string {
	return "connect not exist"
}
func (MsgErrType_ConnectNotExist_Error) Type() uint8 {
	return MsgType_ReplyErrWithConnectNotExist
}

var (
	DEFAULT_MsgErrType_ConnectNotExist_Error = &MsgErrType_ConnectNotExist_Error{}
	DEFAULT_MsgErrType_ApiNotExist_Error     = &MsgErrType_ApiNotExist_Error{}
	DEFAULT_MsgErrType_Timeout_Error         = &MsgErrType_Timeout_Error{}
)

type MsgErrType_ApiNotExist_Error struct {
}

func (*MsgErrType_ApiNotExist_Error) Error() string {
	return "api not exist"
}
func (*MsgErrType_ApiNotExist_Error) Type() uint8 {
	return MsgType_ReplyErrWithApiNotExist
}

type MsgErrType_Timeout_Error struct {
}

func (*MsgErrType_Timeout_Error) Error() string {
	return "timeout"
}
func (*MsgErrType_Timeout_Error) Type() uint8 {
	return MsgType_ReplyErrWithTimeout
}

type MsgErrType_Write_Error struct {
	Err error
}

func (m *MsgErrType_Write_Error) Error() string {
	return m.Err.Error()
}
func (*MsgErrType_Write_Error) Type() uint8 {
	return MsgType_ReplyErrWithWrite
}

type ErrTimeout struct {
	text string
}

func (e *ErrTimeout) Error() string {
	return fmt.Sprintf("timeout %s", e.text)
}

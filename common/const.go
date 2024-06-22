package common

import (
	"errors"
	"fmt"
)

var (
	DEFAULT_ErrMsgLenLimit   = new(ErrMsgLenLimit)
	DEFAULT_ErrMsgCheck      = new(ErrMsgCheck)
	DEFAULT_ErrTimeout       = new(ErrTimeout)
	DEFAULT_ErrConnNotExist  = new(ErrConnNotExist)
	DEFAULT_ErrAuth          = new(ErrAuth)
	DEFAULT_ErrMultipleReply = errors.New("multiple reply are not allowed")
)

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

type ErrConnNotExist struct {
}

func (ErrConnNotExist) Error() string {
	return "connect not exist"
}
func (ErrConnNotExist) Type() uint8 {
	return MsgType_ReplyErrConnNotExist
}

type ErrTimeout struct {
	text string
}

func (e *ErrTimeout) Error() string {
	return fmt.Sprintf("timeout %s", e.text)
}

type ErrAuth struct{}

func (e *ErrAuth) Error() string {
	return "auth fail conn close"
}

func (e *ErrAuth) Type() uint8 {
	return MsgType_PushErrAuthFail
}

package common

import (
	"errors"
	"fmt"
)

var (
	DEFAULT_ErrMsgLenLimit       = new(ErrMsgLenLimit)
	DEFAULT_ErrMsgCheck          = new(ErrMsgCheck)
	DEFAULT_ErrTimeout           = new(ErrTimeout)
	DEFAULT_ErrConnNotExist      = new(ErrConnNotExist)
	DEFAULT_ErrAuthIdExist       = new(ErrAuthIdExist)
	DEFAULT_ErrMultipleReply     = errors.New("multiple reply are not allowed")
	DEFAULT_ErrReplyErrorInvalid = new(ErrReplyErrorInvalid)
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

type ErrAuthIdExist struct{}

func (e *ErrAuthIdExist) Error() string {
	return "auth id exist conn close"
}

func (e *ErrAuthIdExist) Type() uint8 {
	return MsgType_PushErrAuthFailIdExist
}

type ErrReplyErrorInvalid struct {
}

func (e *ErrReplyErrorInvalid) Error() string {
	return "reply error invalid not null but is empty str"
}

type ErrReplyError struct {
	b []byte
}

func (e *ErrReplyError) Error() string {
	return string(e.b)
}

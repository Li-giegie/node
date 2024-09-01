package common

import (
	"fmt"
	"strconv"
)

var (
	DEFAULT_ErrMsgLenLimit       = new(ErrMsgLenLimit)
	DEFAULT_ErrMsgCheck          = new(ErrMsgCheck)
	DEFAULT_ErrTimeout           = new(ErrTimeout)
	DEFAULT_ErrConnNotExist      = new(ErrConnNotExist)
	DEFAULT_ErrReplyErrorInvalid = new(ErrReplyErrorInvalid)
	DEFAULT_ErrDrop              = new(ErrDrop)
)

type ErrMsgLenLimit struct {
}

func (*ErrMsgLenLimit) Error() string {
	return "message length exceeds the limit size 0x00FFFFFF"
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

const limitErrLen = 65533

type ErrReplyErrorInvalid struct {
}

func (e *ErrReplyErrorInvalid) Error() string {
	return "reply error invalid Greater than limit length " + strconv.Itoa(limitErrLen)
}

type ErrReplyError struct {
	b []byte
}

func (e *ErrReplyError) Error() string {
	return string(e.b)
}

type ErrWrite struct {
	err error
}

func (e *ErrWrite) Error() string {
	return e.err.Error()
}

type ErrDrop struct {
}

func (e *ErrDrop) Error() string {
	return "message id not exist or timeout message"
}

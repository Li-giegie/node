package common

import (
	"strconv"
)

var (
	DEFAULT_ErrMsgLenLimit       = new(ErrMsgLenLimit)
	DEFAULT_ErrMsgChecksum       = new(ErrMsgChecksum)
	DEFAULT_ErrTimeout           = new(ErrTimeout)
	DEFAULT_ErrConnNotExist      = new(ErrConnNotExist)
	DEFAULT_ErrReplyErrorInvalid = new(ErrReplyErrorInvalid)
	DEFAULT_ErrDrop              = new(ErrDrop)
)

type ErrMsgLenLimit struct {
}

func (*ErrMsgLenLimit) Error() string {
	return "message length exceeds the limit size"
}

type ErrMsgChecksum struct {
}

func (*ErrMsgChecksum) Error() string {
	return "invalid message checksum error"
}

type ErrConnNotExist struct {
}

func (ErrConnNotExist) Error() string {
	return "connect not exist"
}

type ErrTimeout struct {
	text string
}

func (e *ErrTimeout) Error() string {
	return "timeout"
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
	return "timeout or invalid messages"
}

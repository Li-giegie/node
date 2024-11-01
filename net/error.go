package net

import "strconv"

// NodeError 用于断言是否为Node产生的错误
type NodeError interface {
	NodeError()
}

// ErrReply error参数最大传输字节，65535代表error为nil，65534被保留不可用
const maxErrReplySize = 65533

var (
	DEFAULT_ErrMsgLenLimit              = new(ErrMsgLenLimit)
	DEFAULT_ErrMsgChecksum              = new(ErrMsgChecksum)
	DEFAULT_ErrReplyErrorLengthOverflow = new(ErrReplyErrorLengthOverflow)
	DEFAULT_ErrConnNotExist             = new(ErrConnNotExist)
	DEFAULT_ErrTimeoutMsg               = new(ErrTimeoutMsg)
	DEFAULT_ErrWriteYourself            = new(ErrWriteYourself)
	DEFAULT_ErrReplyLimitOnce           = new(ErrReplyLimitOnce)
	DEFAULT_ErrClosedListen             = new(ErrClosedListen)
)

// ErrReplyLimitOnce 限制回复一次错误,多次回复时产生此错误
type ErrReplyLimitOnce struct{}

func (e *ErrReplyLimitOnce) Error() string {
	return "limit reply to one time"
}
func (e *ErrReplyLimitOnce) NodeError() {}

// ErrWriteYourself 发送给自己时就会产生此错误
type ErrWriteYourself struct{}

func (e *ErrWriteYourself) Error() string {
	return "can't send it to yourself"
}

func (e *ErrWriteYourself) NodeError() {}

// ErrReplyErrorLengthOverflow 调用ErrReply方法时产生，error超过最大65533自己数限制
type ErrReplyErrorLengthOverflow struct{}

func (e *ErrReplyErrorLengthOverflow) Error() string {
	return "ErrReply method Length overflow, maximum length " + strconv.Itoa(maxErrReplySize)
}
func (e *ErrReplyErrorLengthOverflow) NodeError() {}

type ErrMsgLenLimit struct {
}

// ErrMsgLenLimit 消息长度超过节点限制大小
func (*ErrMsgLenLimit) Error() string {
	return "message length exceeds the limit size"
}

func (e *ErrMsgLenLimit) NodeError() {}

// ErrMsgChecksum 消息校验和错误
type ErrMsgChecksum struct{}

func (*ErrMsgChecksum) Error() string {
	return "invalid message checksum error"
}

func (e *ErrMsgChecksum) NodeError() {}

// ErrConnNotExist 节点不存在错误
type ErrConnNotExist struct{}

func (*ErrConnNotExist) Error() string {
	return "connect not exist"
}

func (e *ErrConnNotExist) NodeError() {}

// ErrReplyError 调用ErrReply回复时，得到的响应error为该错误
type ErrReplyError struct {
	b []byte
}

func (e *ErrReplyError) Error() string {
	return string(e.b)
}

func (e *ErrReplyError) NodeError() {}

// ErrTimeoutMsg ErrHandle中出现的错误类型
type ErrTimeoutMsg struct{}

func (e *ErrTimeoutMsg) Error() string {
	return "timeout or invalid messages"
}

func (e *ErrTimeoutMsg) NodeError() {}

type ErrClosedListen struct{}

func (c ErrClosedListen) Error() string {
	return "closed network connection"
}

func (c ErrClosedListen) NodeError() {}

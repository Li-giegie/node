package net

type NodeError []byte

func (n NodeError) Error() string {
	return string(n)
}

func (n NodeError) NodeError() {}

var (
	ErrChecksumInvalid  = NodeError("checksum invalid")
	ErrWriteMsgYourself = NodeError("can't send it to yourself")
	ErrMultipleResponse = NodeError("A request can only be responded to once")
	ErrInvalidResponse  = NodeError("invalid response")
	ErrLengthOverflow   = NodeError("length overflow")
	ErrNodeNotExist     = NodeError("node not exist")
)

package net

type NodeError []byte

func (n NodeError) Error() string {
	return string(n)
}

func (n NodeError) NodeError() {}

var (
	ErrChecksum         = NodeError("checksum invalid")
	ErrMaxMsgLen        = NodeError("maximum message length limit is exceeded")
	ErrNodeNotExist     = NodeError("node not exist")
	ErrWriteMsgYourself = NodeError("can't send it to yourself")
	ErrOnce             = NodeError("limit reply to one time")
)

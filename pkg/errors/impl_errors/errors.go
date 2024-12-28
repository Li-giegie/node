package impl_errors

type NodeError []byte

func (n NodeError) Error() string {
	return string(n)
}

func (n NodeError) NodeError() {}

var (
	ErrChecksumInvalid     = NodeError("checksum invalid")
	ErrWriteMsgYourself    = NodeError("can't send it to yourself")
	ErrMultipleResponse    = NodeError("A request can only be responded to once")
	ErrInvalidResponse     = NodeError("invalid response")
	ErrLengthOverflow      = NodeError("length overflow")
	ErrNodeNotExist        = NodeError("node not exist")
	ConfigMaxConnSleepErr  = NodeError("config.MaxConnSleep Must be greater than 0")
	BridgeRemoteIdExistErr = NodeError("Bridge error: remote id exist")
	AcceptDeniedErr        = NodeError("AcceptCallback denied the connection establishment")
	MultipleConfigErr      = NodeError("config accepts only one parameter")
)

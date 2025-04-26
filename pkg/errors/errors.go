package errors

type NodeError interface {
	Error() string
	NodeError()
}

type Error []byte

func (n Error) Error() string {
	return string(n)
}

func (n Error) NodeError() {}

var (
	ErrChecksumInvalid     = Error("checksum invalid")
	ErrWriteMsgYourself    = Error("can't send it to yourself")
	ErrMultipleResponse    = Error("A request can only be responded to once")
	ErrInvalidResponse     = Error("invalid response")
	ErrLengthOverflow      = Error("length overflow")
	ErrNodeNotExist        = Error("node not exist")
	BridgeRemoteIdExistErr = Error("Bridge error: remote id exist")
)

func New(s string) error {
	return Error([]byte(s))
}

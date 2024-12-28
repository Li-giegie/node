package errors

type NodeError interface {
	Error() string
	NodeError()
}

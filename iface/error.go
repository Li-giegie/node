package iface

// NodeError 用于断言是否为Node产生的错误
type NodeError interface {
	NodeError()
	Error() string
}

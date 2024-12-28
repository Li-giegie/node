package writequeue

type WriteQueue interface {
	Write(b []byte) (n int, err error)
	Close() error
}

package common

import (
	"io"
)

func NewWriteQueue(writer io.Writer, queueSize, bufferSize int) *WriterQueue {
	w := &WriterQueue{
		Writer: writer,
		queue:  make(chan []byte, queueSize),
	}
	go w.start(bufferSize)
	return w
}

type WriterQueue struct {
	io.Writer
	err   error
	queue chan []byte
}

func (w *WriterQueue) start(bufferSize int) {
	buf := make([]byte, 0, bufferSize)
	for b := range w.queue {
		if w.err != nil {
			return
		} else if len(w.queue) == 0 || len(b) >= bufferSize {
			if len(buf) > 0 {
				_, w.err = w.Writer.Write(append(buf, b...))
			} else {
				_, w.err = w.Writer.Write(b)
			}
			buf = buf[:0]
		} else {
			buf = append(buf, b...)
		}
	}
}

func (w *WriterQueue) Write(b []byte) (n int, err error) {
	defer func() { recover() }()
	if w.err != nil {
		return 0, err
	}
	w.queue <- b
	return 0, w.err
}

func (w *WriterQueue) Freed() {
	defer func() { recover() }()
	close(w.queue)
}

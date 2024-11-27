package net

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
	buf := make([]byte, bufferSize)
	var size int
	for b := range w.queue {
		if w.err != nil {
			return
		}
		if size+len(b) >= bufferSize || len(w.queue) == 0 {
			if size > 0 {
				if _, w.err = w.Writer.Write(buf[:size]); w.err != nil {
					return
				}
				size = 0
			}
			_, w.err = w.Writer.Write(b)
		} else {
			copy(buf[size:], b)
			size += len(b)
		}
	}
}

func (w *WriterQueue) Write(b []byte) (n int, err error) {
	if w.err != nil {
		return 0, err
	}
	defer func() { recover() }()
	w.queue <- b
	return len(b), w.err
}

func (w *WriterQueue) Freed() {
	defer func() { recover() }()
	close(w.queue)
}

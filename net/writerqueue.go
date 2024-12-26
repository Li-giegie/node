package net

import (
	"io"
)

func NewWriteQueue(w io.WriteCloser, queueSize, bufferSize int) io.WriteCloser {
	if queueSize <= 1 || bufferSize < 64 {
		return w
	}
	wq := &WriterQueue{
		w:     w,
		queue: make(chan []byte, queueSize),
	}
	go wq.start(bufferSize)
	return wq
}

type WriterQueue struct {
	w     io.WriteCloser
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
				if _, w.err = w.w.Write(buf[:size]); w.err != nil {
					return
				}
				size = 0
			}
			_, w.err = w.w.Write(b)
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

func (w *WriterQueue) Close() error {
	defer func() { recover() }()
	close(w.queue)
	return w.w.Close()
}

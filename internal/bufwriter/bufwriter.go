package bufwriter

import (
	"errors"
	"io"
	"sync/atomic"
)

func NewWriter(w io.Writer, queueCap, bufferCap int) *Writer {
	return &Writer{
		Writer:    w,
		queueCap:  queueCap,
		bufferCap: bufferCap,
	}
}

const (
	stateClose uint32 = iota
	stateStart
)

type Writer struct {
	io.Writer
	err       error
	queue     chan []byte
	queueCap  int
	bufferCap int
	state     uint32
}

func (w *Writer) Start() {
	if w.state == stateStart {
		panic("writer queue already started")
	}
	w.state = stateStart
	w.queue = make(chan []byte, w.queueCap)
	buf := make([]byte, w.bufferCap)
	size := 0
	go func() {
		for b := range w.queue {
			if w.err != nil {
				continue
			}
			if atomic.LoadUint32(&w.state) == stateClose {
				w.err = ErrClosed
				continue
			}
			if size+len(b) >= w.bufferCap {
				if size > 0 {
					if _, w.err = w.Writer.Write(buf[:size]); w.err != nil {
						continue
					}
					size = 0
				}
				if len(b) >= w.bufferCap {
					_, w.err = w.Writer.Write(b)
				} else {
					copy(buf[size:], b)
					size += len(b)
				}
			} else {
				copy(buf[size:], b)
				size += len(b)
			}
			if len(w.queue) == 0 && size > 0 {
				_, w.err = w.Writer.Write(buf[:size])
				size = 0
			}
		}
	}()
}

var ErrClosed = errors.New("writer queue closed")

func (w *Writer) Write(b []byte) (n int, err error) {
	if w.err != nil {
		return 0, w.err
	}
	if atomic.LoadUint32(&w.state) == stateClose {
		return 0, ErrClosed
	}
	w.queue <- b
	return len(b), w.err
}

func (w *Writer) Close() error {
	if atomic.CompareAndSwapUint32(&w.state, stateStart, stateClose) {
		close(w.queue)
		if len(w.queue) > 0 {
			for range w.queue {
			}
		}
		return nil
	} else {
		return errors.New("writer queue already closed")
	}
}

func (w *Writer) Error() error {
	return w.err
}

func (w *Writer) State() uint32 {
	return w.state
}

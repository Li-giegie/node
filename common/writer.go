package common

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

func NewWriter(w io.Writer, size int) *Writer {
	return &Writer{
		buf:    make([]byte, 0, size),
		Writer: w,
		cap:    size,
	}
}

// Writer 针对net.Conn，实现高性能Write，减少并发情况下频繁系统调用开销，支持并发调用
type Writer struct {
	buf      []byte
	refCount int32
	err      error
	cap      int
	io.Writer
	sync.Mutex
}

func (c *Writer) Write(b []byte) (n int, err error) {
	if c.err != nil {
		return 0, err
	}
	lb := len(b)
	if lb == 0 {
		return 0, nil
	} else if lb >= c.cap {
		return c.Writer.Write(b)
	}
	// 引用计数 用于统计有多少协程调用了Write方法 这里是已原子增加1 这里会有多个协程能访问到
	atomic.AddInt32(&c.refCount, 1)
	// 加锁同一时间内仅有一个协程进入
	c.Lock()
	// 如果写入的数据长度+缓存区历史数据大于缓冲区容量则调用底层Write发送数据并且引用计数器值-1
	if lb+len(c.buf) >= c.cap {
		n, err = c.Writer.Write(append(c.buf, b...))
		if err != nil {
			c.err = err
		} else {
			c.buf = c.buf[:0]
		}
		atomic.AddInt32(&c.refCount, -1)
		c.Unlock()
		return
	}
	c.buf = append(c.buf, b...)
	// 如果对计数器 -1 =0 则认为一组协程可能结束 触发调用底层写入
	if atomic.AddInt32(&c.refCount, -1) == 0 {
		n, err = c.Writer.Write(c.buf)
		if err != nil {
			c.err = err
		} else {
			c.buf = c.buf[:0]
		}
		c.Unlock()
		return
	}
	c.Unlock()
	return lb, nil
}

var traceAllWriteCount, traceOverflowWriteCount, traceLastGoroutineWriteCount, traceWriteBufferCount uint32

func PrintTrace() {
	fmt.Printf("allWriteCount %d OverflowWriteCount %d LastGoroutineWriteCount %d WriteBufferCount %d\n", traceAllWriteCount, traceOverflowWriteCount, traceLastGoroutineWriteCount, traceWriteBufferCount)
}

// TraceWrite 追踪调用情况，在common/conn.go文件中connection.write方法中调用该方法，使用PrintTrace查看结果
func (c *Writer) TraceWrite(b []byte) (n int, err error) {
	atomic.AddUint32(&traceAllWriteCount, 1)
	if c.err != nil {
		return 0, err
	}
	lb := len(b)
	if lb == 0 {
		return 0, nil
	} else if lb >= c.cap {
		return c.Writer.Write(b)
	}
	// 引用计数 用于统计有多少协程调用了Write方法 这里是已原子增加1 这里会有多个协程能访问到
	atomic.AddInt32(&c.refCount, 1)
	// 加锁同一时间内仅有一个协程进入
	c.Lock()
	// 如果写入的数据长度+缓存区历史数据大于缓冲区容量则调用底层Write发送数据并且引用计数器值-1
	if lb+len(c.buf) >= c.cap {
		traceOverflowWriteCount++
		n, err = c.Writer.Write(append(c.buf, b...))
		if err != nil {
			c.err = err
		} else {
			c.buf = c.buf[:0]
		}
		atomic.AddInt32(&c.refCount, -1)
		c.Unlock()
		return
	}
	c.buf = append(c.buf, b...)
	// 如果对计数器 -1 =0 则认为一组协程可能结束 触发调用底层写入
	if atomic.AddInt32(&c.refCount, -1) == 0 {
		traceLastGoroutineWriteCount++
		n, err = c.Writer.Write(c.buf)
		if err != nil {
			c.err = err
		} else {
			c.buf = c.buf[:0]
		}
		c.Unlock()
		return
	}
	traceWriteBufferCount++
	c.Unlock()
	return lb, nil
}

package test

import (
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

func BenchmarkAppendA(b *testing.B) {
	buf := make([]byte, 512)
	for i := 0; i < b.N; i++ {
		appendA(buf)
	}
}
func BenchmarkAppendB(b *testing.B) {
	buf := make([]byte, 512)
	for i := 0; i < b.N; i++ {
		appendB(buf)
	}
}
func BenchmarkAppendC(b *testing.B) {
	buf := make([]byte, 512)
	for i := 0; i < b.N; i++ {
		appendC(buf)
	}
}

func BenchmarkAppendC2(b *testing.B) {
	buf := make([]byte, 512)
	for i := 0; i < b.N; i++ {
		appendC2(buf)
	}
}

func BenchmarkAppendGoA(b *testing.B) {
	buf := make([]byte, 512)
	w := sync.WaitGroup{}
	for i := 0; i < b.N; i++ {
		w.Add(1)
		go func() {
			appendA(buf)
			w.Done()
		}()
	}
	w.Wait()
}
func BenchmarkAppendGoB(b *testing.B) {
	buf := make([]byte, 512)
	w := sync.WaitGroup{}
	for i := 0; i < b.N; i++ {
		w.Add(1)
		go func() {
			appendB(buf)
			w.Done()
		}()
	}
	w.Wait()
}

func BenchmarkAppendGoC(b *testing.B) {
	buf := make([]byte, 512)
	w := sync.WaitGroup{}
	for i := 0; i < b.N; i++ {
		w.Add(1)
		go func() {
			appendC(buf)
			w.Done()
		}()
	}
	w.Wait()
}

func BenchmarkAppendGoC2(b *testing.B) {
	buf := make([]byte, 512)
	w := sync.WaitGroup{}
	for i := 0; i < b.N; i++ {
		w.Add(1)
		go func() {
			appendC2(buf)
			w.Done()
		}()
	}
	w.Wait()
}

var n = []byte{1, 2, 3, 4}

type spinLock uint32

const maxBackoff = 16

func (sl *spinLock) Lock() {
	backoff := 1
	for !atomic.CompareAndSwapUint32((*uint32)(sl), 0, 1) {
		// 指数退避算法
		for i := 0; i < backoff; i++ {
			runtime.Gosched()
			// runtime.Gosched()，用于让出CPU时间片，让出当前goroutine的执行权限，
			// 调度器安排其它等待的任务运行，并在下次某个时候从该位置恢复执行。
		}
		if backoff < maxBackoff {
			backoff <<= 1
		}
	}
}

func (sl *spinLock) Unlock() {
	atomic.StoreUint32((*uint32)(sl), 0)
}

func appendA(b []byte) {
	Write(append(n, b...))
}

func appendB(b []byte) {
	buf := make([]byte, 4+len(b))
	copy(buf, n)
	copy(buf[4:], b)
	Write(buf)
}

var l sync.Mutex

func appendC(b []byte) {
	l.Lock()
	Write(n)
	Write(b)
	l.Unlock()
}

var l3 spinLock

func appendC2(b []byte) {
	l3.Lock()
	Write(n)
	Write(b)
	l3.Unlock()
}

var l2 sync.Mutex

func Write(b []byte) {
	l2.Lock()
	io.Discard.Write(b)
	l2.Unlock()
}

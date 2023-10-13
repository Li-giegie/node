package node

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// 协程池结构体
type Pool struct {
	arg        chan interface{}
	handleFunc func(arg interface{})
	wg         sync.WaitGroup
}

// 创建协程池
func NewPool(numWorkers int, handle func(interface{})) *Pool {
	p := new(Pool)
	p.arg = make(chan interface{}, numWorkers)
	p.handleFunc = handle
	p.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go p.worker()
	}
	return p
}

// 添加任务到协程池
func (p *Pool) AddTask(task interface{}) {
	p.arg <- task
}

var index int32

// 工作协程
func (p *Pool) worker() {
	id := atomic.AddInt32(&index, 1)
	fmt.Println("start worker ", id)
	for task := range p.arg {
		p.handleFunc(task)
	}
	p.wg.Done()
}

// 等待所有任务完成
func (p *Pool) Wait() {
	close(p.arg)
	p.wg.Wait()
}

func TestTask(t *testing.T) {
	//task := NewPool(12, func(i interface{}) {
	//	//fmt.Println(i)
	//	time.Sleep(time.Millisecond * 10)
	//})
	//
	//fmt.Println(len(task.arg))
	//t1 := time.Now()
	//for i := 0; i < 1000; i++ {
	//	task.arg <- i
	//}
	//
	//task.Wait()
	//fmt.Println(time.Since(t1))
	//
	//t1 = time.Now()
	//for i := 0; i < 100000; i++ {
	//	w.Add(1)
	//	go func() {
	//		time.Sleep(time.Millisecond * 1)
	//		w.Done()
	//	}()
	//}
	//w.Wait()
	//fmt.Println(time.Since(t1))
	//
	//t1 = time.Now()
	//for i := 0; i < 100000; i++ {
	//	w.Add(1)
	//	go func() {
	//		time.Sleep(time.Millisecond * 1)
	//		w.Done()
	//	}()
	//}
	//w.Wait()
	//fmt.Println(time.Since(t1))
	//
	//defer ants.Release()
	//p, _ := ants.NewPool(1)
	//pp, _ := ants.NewPoolWithFunc(1, func(i interface{}) {
	//
	//})
	//pp.Invoke()
	//t1 = time.Now()
	//for i := 0; i < 100000; i++ {
	//	w.Add(1)
	//	_ = ants.Submit(func() {
	//		time.Sleep(time.Millisecond * 100)
	//		w.Done()
	//	})
	//}
	//w.Wait()
	//fmt.Println(time.Since(t1))
	//
	//an, _ := ants.NewPool(100000)
	//defer an.Release()
	//t1 = time.Now()
	//for i := 0; i < 100000; i++ {
	//	w.Add(1)
	//	_ = an.Submit(func() {
	//		time.Sleep(time.Millisecond * 100)
	//		w.Done()
	//	})
	//}
	//w.Wait()
	//fmt.Println(time.Since(t1))

}

package goroutine_manager

import (
	"fmt"
	"testing"
	"time"
)

func TestGoroutineManager(t *testing.T) {
	a := make(chan interface{}, 5)
	NewGoroutineManager(a, func(arg interface{}) {
		panic(any("asd"))
	}).Run()

	time.Sleep(time.Second * 5)
	fmt.Println("end -----")
}

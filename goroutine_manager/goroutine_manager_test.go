package node

import (
	"fmt"
	"testing"
)

func TestGoroutineManager(t *testing.T) {
	a := make(chan interface{}, 5)
	fmt.Println(len(a), cap(a))
}

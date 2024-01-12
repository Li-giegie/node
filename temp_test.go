package node

import (
	"fmt"
	"sync"
	"testing"
)

var p sync.Pool = sync.Pool{
	New: func() any {
		fmt.Println("重新分配")
		return new(message)
	},
}

var index uint64

func BenchmarkName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Checksum(make([]byte, 1024))
	}
}

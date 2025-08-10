package bufwriter

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestWriter(t *testing.T) {
	var buf bytes.Buffer
	wq := NewWriter(&buf, 10, 128)
	wq.Start()
	for i := 0; i < 10; i++ {
		_, err := wq.Write([]byte(strconv.Itoa(i)))
		if err != nil {
			t.Error(err)
			return
		}
	}
	//wq.Close()
	time.Sleep(time.Millisecond * 1000)
	fmt.Println(buf.String())
	fmt.Println(wq.Error())
}

func BenchmarkWriter(b *testing.B) {
	wq := NewWriter(io.Discard, 10, 128)
	wq.Start()
	for i := 0; i < b.N; i++ {
		wq.Write([]byte(strconv.Itoa(i)))
	}
}

func BenchmarkWriterGo(b *testing.B) {
	wq := NewWriter(io.Discard, 10, 128)
	wq.Start()
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := wq.Write([]byte(strconv.Itoa(i)))
			if err != nil {
				b.Error(err)
				return
			}
		}(i)
	}
	wg.Wait()
}

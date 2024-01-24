package node

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestTcpClient(t *testing.T) {
	fmt.Println(100000)
	addr := "39.101.193.248:8088"
	for i := 0; i < 10; i++ {
		fmt.Println("index ", i)
		//time.Sleep(time.Second * 1)
		go func(j int) {
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer conn.Close()
			var n uint32 = 1 << 31
			if _, err = conn.Write(uint32ToBytes(n)); err != nil {
				fmt.Println(err)
				return
			}
			for {
				buf, err := readAtLeast(conn, 10)
				if err != nil {
					fmt.Println("close id: ", j, err)
					return
				}
				fmt.Println("id: ", j, string(buf))
			}
		}(i)
		continue

	}
	select {}
}

func TestReader(t *testing.T) {
	conn := reader()
	time.Sleep(time.Second)
	r := newReader(conn, 2048)
	for {
		var b = make([]byte, 1000)
		n, err := r.Read(b)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("读取成功：", n)
	}
}

type LimitedReader struct {
	R io.Reader
	N int
}

func newReader(r io.Reader, n int) *LimitedReader {
	return &LimitedReader{
		R: r,
		N: n,
	}
}

func (lr *LimitedReader) Read(p []byte) (int, error) {
	if lr.N <= 0 {
		return 0, io.EOF
	}
	if len(p) > lr.N {
		p = p[:lr.N]
	}
	n, err := lr.R.Read(p)
	lr.N -= n
	return n, err
}

func reader() io.Reader {
	buf := bytes.NewBuffer(nil)
	go func() {
		var i int
		for {
			_, err := buf.Write(make([]byte, 1024))
			if err != nil {
				fmt.Println(err)
			}
			i++
			fmt.Println("write: 1024 index ", i)
			time.Sleep(time.Second * 1)
		}
	}()
	return buf
}

package node

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	var errI = make([]interface{}, 3)
	var err error
	errI[0] = err
	errI[1] = nil
	errI[2] = errors.New("a")
	fmt.Println(errI)
	fmt.Println(errI[0] == nil)
	fmt.Println(errI[1] == nil)
	fmt.Println(errI[2] == nil)
	v, ok := errI[2].(error)
	fmt.Println(v, ok)
}

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

func TestAAA(t *testing.T) {
	p, _ := ants.NewPool(1000)
	for i := 0; i < 10; i++ {
		c := getN()
		p.Submit(func() {
			fmt.Println(c, i)
		})
	}
	time.Sleep(time.Second * 10)
}

var ccc int

func getN() int {
	ccc++
	return ccc
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

func TestN(t *testing.T) {
	f, err := os.OpenFile("1.txt", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()
	fmt.Println(f.WriteString("abcdef\n"))
	fmt.Println(f.WriteString("abcdef\n"))
	fmt.Println(f.WriteString("abcdef\n"))
}

func TestCCC(t *testing.T) {
	http.HandleFunc("/ping", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println("ping---", request.ContentLength)
		writer.WriteHeader(200)
		fmt.Println(writer.Write([]byte("pone")))
	})
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()
	time.Sleep(time.Second * 2)
	var data = `GET /ping HTTP/1.1
Host: localhost:8080
User-Agent: Go-http-client/1.1
Content-Length: 9000

[   data]`
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()
	_, err = conn.Write([]byte(data))
	if err != nil {
		t.Error(err)
		return
	}
	var rbuf = make([]byte, 100)
	for {
		n, err := conn.Read(rbuf)
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Println("receive: ", string(rbuf[:n]))
	}
}

func TestHttpClient(t *testing.T) {
	for {
		fmt.Println(http.Get("http://127.0.0.1:8080/ping"))
		time.Sleep(time.Second)
	}
}

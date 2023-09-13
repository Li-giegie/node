package test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"github.com/Li-giegie/node"
	"io"
	"log"
	"net"
	"strconv"
	"testing"
)

type TestSender struct {
	Api    uint32
	Handle node.HandlerFunc
}

func (s *TestSender) GetApi() uint32 {
	return 1
}

func (s *TestSender) GetHandler() node.HandlerFunc {
	return func(ctx *node.Context) {
		fmt.Println("message: ", ctx.String())
	}
}

type TestRequester struct {
	Api    uint32
	Handle node.HandlerFunc
}

func (s *TestRequester) GetApi() uint32 {
	return 1
}

func (s *TestRequester) GetHandler() node.HandlerFunc {
	return func(ctx *node.Context) {
		fmt.Println("message: ", ctx.String())
		err := ctx.ReplyString("收到！！！")
		if err != nil {
			fmt.Println(err)
		}
	}
}

func TestServer(t *testing.T) {
	srv := node.NewServer(node.DEFAULT_ServerAddress)
	srv.AddRouterHandler(&TestRequester{})
	err := srv.ListenAndServer()
	if err != nil {
		fmt.Println(err)
	}
}

func TestServer2(t *testing.T) {
	listen, err := net.Listen("tcp", "127.0.0.1:2024")
	if err != nil {
		log.Fatalln(err)
	}
	defer listen.Close()

	fmt.Println("listen success ---")
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatalln(err)
		}

		go func() {
			log.Println("connect ---")
			for {
				var buf = make([]byte, 100)
				fmt.Println(len(buf), cap(buf))
				buf, err := readN(conn, 4)
				if err != nil {
					log.Fatalln("read", err)
				}
				n := binary.LittleEndian.Uint32(buf)
				buf, err = readN(conn, int(n))
				if err != nil {
					log.Fatalln("read err -2", err)
				}
				fmt.Println(string(buf))
			}
		}()
	}
}

func readN(r io.Reader, l int) ([]byte, error) {
	var buf = make([]byte, l)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if n == l {
		return buf, nil
	}

	var nn int
	nn += n
	for {
		n, err = r.Read(buf[nn:])
		if err != nil && err != io.EOF {
			return nil, err
		}
		nn += n
		if nn == l {
			return buf, nil
		}
	}

}

var js = 100000000

// 耗时 2.9050577s 效率 request num:[0],reply num:[100000000]
// 耗时 3.8833892s 效率 request num:[0],reply num:[100000000]

// 耗时 3.3710122s 效率 request num:[0],reply num:[100000000]
// 耗时 4.0388354s 效率 request num:[0],reply num:[100000000]
func TestReadN(t *testing.T) {
	var b = bytes.NewBuffer(nil)

	for i := 0; i < js; i++ {
		_, err := b.Write(jeans.Pack([]byte(strconv.Itoa(i))))
		if err != nil {
			log.Fatalln("init error", err)
		}
	}
	fmt.Println("缓冲区完成！", b.Len())
	var c = node.NewCounter()
	for {
		buf, err := readN(b, 4)
		if err != nil {
			c.Debug()
			log.Fatalln("readn err -1", err)
		}
		n := binary.LittleEndian.Uint32(buf)
		buf, err = readN(b, int(n))
		if err != nil {
			c.Debug()
			log.Fatalln(err)
		}
		nn := c.AddReplyNum()
		if nn == uint64(js) {
			break
		}
	}
	c.Debug()
	fmt.Println("-----------")
}
func TestReadFull(t *testing.T) {
	var b = bytes.NewBuffer(nil)
	for i := 0; i < js; i++ {
		_, err := b.Write(jeans.Pack([]byte(strconv.Itoa(i))))
		if err != nil {
			log.Fatalln("init error", err)
		}
	}
	fmt.Println("缓冲区完成！", b.Len())
	var c = node.NewCounter()
	for {
		var lb = make([]byte, 4)
		_, err := io.ReadFull(b, lb)
		if err != nil {
			c.Debug()
			log.Fatalln("readn err -1", err)
		}
		n := binary.LittleEndian.Uint32(lb)
		var buf = make([]byte, n)
		_, err = io.ReadFull(b, buf)
		if err != nil {
			c.Debug()
			log.Fatalln(err)
		}
		nn := c.AddReplyNum()
		if nn == uint64(js) {
			break
		}
	}
	c.Debug()
	fmt.Println("-----------")
}

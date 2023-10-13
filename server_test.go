package node

import (
	"bytes"
	"encoding/binary"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"io"
	"log"
	"strconv"
	"testing"
	"time"
)

type ReqScene struct {
}

func (ReqScene) Hello() []byte {
	return []byte("hello scene1")
}

func (ReqScene) Api() uint32 {
	return 1
}

func (ReqScene) Handler() HandlerFunc {
	return func(ctx *Context) {
		//log.Println("scene1:", ctx.String())
		if len(ctx.Data) != 0 {
			_ = ctx.Write([]byte("scene1 success"))
		}
	}
}

func TestNodeServer(t *testing.T) {
	srv := NewServer(DEFAULT_ServerID, DEFAULT_ServerAddress)
	srv.AddRouterI(ReqScene{})
	srv.Id = DEFAULT_ServerID
	srv.MaxConnectNum = 100000
	srv.ConnectionTimeout = time.Second * 6
	srv.AuthenticationFunc = func(id string, data []byte) (ok bool, reply []byte) {
		return true, []byte("服务器测试认证")
	}

	err := srv.ListenAndServer()
	if err != nil {
		fmt.Println(err)
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
	var c = NewCounter()
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
		//nn := c.AddReplyNum()
		//if nn == uint64(js) {
		//	break
		//}
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
	var c = NewCounter()
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
		//nn := c.AddReplyNum()
		//if nn == uint64(js) {
		//	break
		//}
	}
	c.Debug()
	fmt.Println("-----------")
}

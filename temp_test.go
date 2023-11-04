package node

import (
	"bytes"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"github.com/tidwall/evio"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

func TestPack(t *testing.T) {
	buf := jeans.Pack(nil)
	fmt.Println(buf, len(buf))
	r := bytes.NewBuffer(buf)
	buf, err := jeans.Unpack(r)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(buf, len(buf))

}

func TestSyncMap(t *testing.T) {
	var m sync.Map
	m.Store(1, 1)
	m.Store("p", 2)
	fmt.Println(m.Load(1))
	fmt.Println(m.Load("p"))
}

func TestGetCPUCore(t *testing.T) {
	fmt.Println(runtime.NumCPU())
}

func TestSetData(t *testing.T) {
	//m := NewMsg()
	//m._type = MsgType_Resp
	//m.Data = []byte("hello")
	//mb := m.Marshal()
	//fmt.Println(NewMsgWithUnmarshal(mb), m.Data)
	//
	//m.localId = "local"
	//m.remoteId = "remote"
	//m2 := NewMsgWithUnmarshal(m.buf)
	//
	//fmt.Println(NewMsgWithUnmarshal(m2.buf))
	//
	//fmt.Println("000  ", m2.buf)
	//fmt.Println("111  ", m2.Data)
	//fmt.Println("1111 ", m2.String())
	//
	//fmt.Println("222  ", m2.buf)
	//fmt.Println("333 ", m2.Data)
	//fmt.Println("222  ", m2.String())
}

// BenchmarkSetData_Msg-12    	100000000	        11.23 ns/op
func BenchmarkSetData_Msg(b *testing.B) {
	//m := NewMsg()
	//m.buf = []byte{1, 2, 3, 4, 5}
	//for i := 0; i < b.N; i++ {
	//	//m.setData([]byte{0, 1, 2, 3})
	//	//m.setType(MsgType_Resp)
	//}
}

// BenchmarkMarshal_Msg-12    	18630300	        58.44 ns/op
func BenchmarkMarshal_Msg(b *testing.B) {
	//m := newMsg()
	//for i := 0; i < b.N; i++ {
	//	m.marshalV1()
	//}
}

func TestServer(t *testing.T) {
	var count int32
	listen, err := net.Listen("tcp", DEFAULT_ServerAddress)
	if err != nil {
		t.Error(err)
		return
	}
	defer listen.Close()

	fmt.Println("listen success ---")
	for {
		conn, err := listen.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		count++
		fmt.Println("server count", count)
		go func(_conn net.Conn) {
			for {
				_, err := jeans.Unpack(_conn)
				if err != nil {
					atomic.AddInt32(&count, -1)
					return
				}
				_, err = _conn.Write(jeans.Pack([]byte("receive success")))
				if err != nil {
					atomic.AddInt32(&count, -1)
					return
				}
				//fmt.Println("read ", string(buf))
			}
		}(conn)
	}
}

func TestEVIO(t *testing.T) {
	var events evio.Events
	events.NumLoops = 4
	events.Data = func(c evio.Conn, in []byte) (out []byte, action evio.Action) {
		//fmt.Println(string(in))
		out = in
		return
	}
	if err := evio.Serve(events, "tcp://localhost:2023"); err != nil {
		t.Error(err)
	}
}

func TestAAAA(t *testing.T) {
	var c = make(chan *int)
	var i interface{} = c
	close(c)
	v, ok := i.(chan *int)
	fmt.Println(6, v, ok, v == nil, c == nil, c)
	v <- new(int)

}

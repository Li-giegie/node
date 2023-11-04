package node

import (
	"fmt"
	"testing"
)

//goos: windows
//goarch: amd64
//pkg: github.com/Li-giegie/node
//cpu: AMD Ryzen 5 5600H with Radeon Graphics

func TestUn_MarshalV1(t *testing.T) {
	m := newMsgWithReq(1, []byte{65, 66, 67})
	buf := m.marshalV1()
	m2 := newMsgWithUnmarshalV1(buf)
	fmt.Println(m2.String())
}

func TestUn_MarshalV2(t *testing.T) {
	m := newMsgWithReq(1, []byte{65, 66, 67})
	buf := m.marshalV2()
	m2 := newMsgWithUnmarshalV2(buf)
	fmt.Println(m2.String())
}

func BenchmarkNewMsgWithReqV2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := newMsgWithReq(uint32(i), []byte{})
		m.recycle()
	}
}

// BenchmarkMarshalMsgWithReqV1-12         24016234                47.46 ns/op
func BenchmarkMarshalMsgWithReqV1(b *testing.B) {
	m := newMsgWithReq(1, []byte{})
	for i := 0; i < b.N; i++ {
		m.marshalV1()
	}
}

// BenchmarkUnmarshalMsgWithReqV1-12       43635252                29.63 ns/op
func BenchmarkUnmarshalMsgWithReqV1(b *testing.B) {
	buf := newMsgWithReq(1, []byte{}).marshalV1()
	for i := 0; i < b.N; i++ {
		m2 := newMsgWithUnmarshalV1(buf)
		m2.recycle()
	}
}

// BenchmarkMarshalMsgWithReqV2-12         26975080                45.15 ns/op
func BenchmarkMarshalMsgWithReqV2(b *testing.B) {
	m := newMsgWithReq(1, []byte{})
	for i := 0; i < b.N; i++ {
		m.marshalV2()
	}
}

// BenchmarkUnmarshalMsgWithReqV2-12       49385358                25.97 ns/op
func BenchmarkUnmarshalMsgWithReqV2(b *testing.B) {
	buf := newMsgWithReq(1, []byte{}).marshalV2()
	for i := 0; i < b.N; i++ {
		m2 := newMsgWithUnmarshalV2(buf)
		m2.recycle()
	}
}

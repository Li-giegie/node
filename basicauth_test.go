package node

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestBasicAuthReq_Send(t *testing.T) {
	w := bytes.NewBuffer(nil)
	fmt.Println(defaultBasicReq.Send(w, 0, 0xffffffff, []byte("123"), 0))
	fmt.Println(defaultBasicReq.Receive(w, time.Second))
	fmt.Println(w.Bytes())
}

func TestBasicAuthResp_Send(t *testing.T) {
	w := bytes.NewBuffer(nil)
	defaultBasicResp.Send(w, true, "777")
	fmt.Println(w.Bytes())
	fmt.Println(defaultBasicResp.Receive(w, time.Second))
	fmt.Println(w.Bytes())
}

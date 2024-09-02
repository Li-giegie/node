package node

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestAuth(t *testing.T) {
	req := new(basicAuthReq)
	buf := bytes.NewBuffer(nil)
	req.Send(buf, 1, []byte("123213"))
	req.Send(buf, 10, []byte("2345"))
	fmt.Println("req receive")
	for {
		id, accessKey, err := req.Receive(buf, time.Second)
		if err != nil {
			fmt.Println(err)
			break
		}
		fmt.Println(id, string(accessKey))
	}

	resp := new(basicAuthResp)
	resp.Send(buf, true, "")
	resp.Send(buf, true, "success")
	resp.Send(buf, false, "")
	resp.Send(buf, true, "error")

	fmt.Println("resp receive")
	for {
		permit, msg, err := resp.Receive(buf, time.Second)
		if err != nil {
			fmt.Println(err)
			break
		}
		fmt.Println(permit, msg)
	}
}

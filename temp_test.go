package node

import (
	"bytes"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"testing"
)

func TestPack(t *testing.T) {
	buf := jeans.Pack(nil)
	r := bytes.NewBuffer(buf)
	buf, err := jeans.Unpack(r)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(buf, len(buf))
}

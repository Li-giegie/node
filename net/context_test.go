package net

import (
	"fmt"
	"testing"
)

func TestReplyError(t *testing.T) {
	a := []byte{1, 2, 3}
	var b []byte
	b = append(b, a...)
	fmt.Println(b)
	b[0] = 0
	fmt.Println(b, a)

}

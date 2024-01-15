package node

import (
	"fmt"
	"testing"
	"time"
)

func TestSession(t *testing.T) {
	s := newSessionCache(time.Second*3, time.Second)
	fmt.Println(s.create("a"))
	time.Sleep(time.Second)
	fmt.Println(s.create("b"))
	time.Sleep(time.Second * 5)
	s.stopTimeoutCheck()
	select {}
}

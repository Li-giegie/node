package node

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"sync"
	"testing"
	"time"
)

func TestPack(t *testing.T) {
	var l sync.RWMutex
	var m = map[string]string{
		"":  "",
		"1": "",
		"2": "",
	}
	l.RLock()
	for _, s2 := range m {
		l.Lock()
		fmt.Println(s2)
		l.Unlock()
	}
	l.RUnlock()
}

func TestAnts(t *testing.T) {
	p, _ := ants.NewPool(10)
	p.Tune(20)
	for i := 0; i < 100; i++ {
		n := i
		err := p.Submit(func() {
			time.Sleep(time.Second)
			fmt.Println(n)
		})
		if err != nil {
			fmt.Println(err)
		}
	}
	fmt.Println("cap ", p.Cap())
	fmt.Println("running ", p.Running())
	p.Waiting()

	p.Release()

}

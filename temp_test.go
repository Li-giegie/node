package node

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"log"
	"testing"
	"time"
)

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

func TestTimeOut(t *testing.T) {
	dt := time.Duration(time.Now().Unix()) * time.Second
	for {
		time.Sleep(time.Second)
		if checkUpTimeOut(dt, time.Second*5) {
			log.Println(int64(dt.Seconds()), time.Second*5, time.Now().Unix())
		}
	}

}

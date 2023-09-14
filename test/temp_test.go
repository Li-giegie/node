package test

import (
	"bytes"
	"fmt"
	jeans "github.com/Li-giegie/go-jeans"
	"runtime"
	"sync"
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

func TestWorkerProcess(t *testing.T) {

}

type F interface {
	Say()
}

type Father struct {
	c Child
}

func (a Father) Say() {
	fmt.Println("father")
}

type Child struct {
	F
}

func (b Child) Run() {
	fmt.Println("child")
	b.Say()
}

func TestAB(t *testing.T) {
	var f Father
	c := f.c

	c.F = f

	c.Run()
}

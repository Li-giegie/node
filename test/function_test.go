package test

import (
	"fmt"
	utils "github.com/Li-giegie/go-utils"
	"math/rand"
	"strconv"
	"testing"
)

type iNode interface {
	next() (iNode, bool)
}

type varNode struct {
	name  *string
	value string
	iNode
}

func (n *varNode) next() (iNode, bool) {
	return &varNode{
		name:  nil,
		value: " ",
		iNode: &varNode{
			name:  nil,
			value: "",
			iNode: nil,
		},
	}, true
}

var worldTable = map[string]*iNode{}

type Function struct {
	Name    string
	InArgs  []interface{}
	OutArgs []interface{}
	Context string
}

func newFunc(name string, InArgs, OutArgs []interface{}) *Function {
	return &Function{
		Name:    name,
		InArgs:  InArgs,
		OutArgs: OutArgs,
	}
}

func (f *Function) Run() {

}

func (f *Function) Return() func([]interface{}) []interface{} {
	return func(i []interface{}) []interface{} {
		return nil
	}
}

func TestMap(t *testing.T) {
	var n uint32 = 1 << 26
	var m map[string]struct{}
	fmt.Println(n)
	utils.Run(1, func() {
		m = make(map[string]struct{}, n)
	}).Debug()

	var i uint32

	for i = 0; i <= n; i++ {
		m[strconv.Itoa(int(i))] = struct{}{}
		if i%10000000 == 0 {
			fmt.Println(n - i)
		}
	}

	utils.Run(1, func() {
		fmt.Println(len(m))
	}).Debug()

	for i = 0; i < 100; i++ {
		utils.Run(1, func() {
			v, ok := m[strconv.Itoa(rand.Intn(int(n)))]
			fmt.Println(v, ok)
		}).Debug()
	}
}

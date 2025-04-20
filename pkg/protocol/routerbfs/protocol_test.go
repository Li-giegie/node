package routerbfs

import (
	"fmt"
	"testing"
)

func TestName(t *testing.T) {

	sync1 := SyncMsg{
		Id:         0,
		SubNodeNum: 0,
		Hash:       0,
	}
	sync2 := &SyncMsg{
		Id:         0,
		SubNodeNum: 0,
		Hash:       0,
	}
	fmt.Println(sync1 == *sync2)
}

func checkAndReplace(num int64, arr []int64) bool {
	minIndex := 0
	minVal := arr[0]
	for i, v := range arr {
		if num == v {
			return false
		}
		if v < minVal {
			minVal = v
			minIndex = i
		}
	}
	if num < minVal {
		return false
	}
	arr[minIndex] = num
	return true
}

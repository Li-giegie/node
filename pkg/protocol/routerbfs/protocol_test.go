package routerbfs

import (
	"fmt"
	"testing"
)

func TestName(t *testing.T) {
	nums := []int64{0, 0, 0, 0, 0}
	fmt.Println(checkAndReplace(3, nums), nums)
	fmt.Println(checkAndReplace(1, nums), nums)
	fmt.Println(checkAndReplace(5, nums), nums)
	fmt.Println(checkAndReplace(2, nums), nums)
	fmt.Println(checkAndReplace(6, nums), nums)
	fmt.Println(checkAndReplace(7, nums), nums)
	fmt.Println(checkAndReplace(3, nums), nums)
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

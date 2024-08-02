package main

import "fmt"

// N n
const N = 10

type arr [N]int

func main() {
	a := arr{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	fmt.Println(a)
	a.insert(-1, 100)
	fmt.Println(a)
}

func (a *arr) insert(k int, v int) bool {
	// 超出数组范围的返回false
	if k >= N || k < 0 {
		return false
	}
	for i := N - 1; i > k; i-- {
		a[i] = a[i-1]
	}

	a[k] = v
	return true
}

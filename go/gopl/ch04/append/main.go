package main

import "fmt"

func main() {
	var x, y []int
	for i := 0; i < 10; i++ {
		y = appendInt(x, i)
		fmt.Printf("i=%d len=%d cap=%d\t%v\n", i, len(y), cap(y), y)
		x = y
	}
}

func appendInt(x []int, y ...int) []int {
	var z []int
	zLen := len(x) + len(y)
	if zLen <= cap(x) {
		z = x[:zLen]
	} else {
		zCap := zLen
		if zCap < 2*len(x) {
			zCap = 2 * len(x)
		}

		z = make([]int, zLen, zCap)
		copy(z, x)
	}
	copy(z[len(x):], y)
	return z
}

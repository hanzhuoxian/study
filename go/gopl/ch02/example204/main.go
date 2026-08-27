package main

import "fmt"

func PopCount(x uint64) int {
	var t uint64 = 1
	var num int
	for i := 0; i < 64; i++ {
		a := x & (t << i)
		fmt.Println(t, i, a)
		if a > 0 {
			num++
		}
	}
	return num
}

func main() {
	fmt.Println(PopCount(2))
}

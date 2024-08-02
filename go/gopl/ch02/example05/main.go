package main

import "fmt"

func PopCount(x uint64) int {
	var num int
	for i := 0; i < 64; i++ {
		fmt.Println(x, x-1, x&(x-1))
	}
	return num
}

func main() {
	fmt.Println(PopCount(2))
}

package main

import "fmt"

func main() {
	fmt.Println()
	sq := square()
	for i := 0; i < 5; i++ {
		fmt.Println(sq())
	}
}

func square() func() int {
	var x int
	return func() int {
		x++
		return x * x
	}
}

package main

import "fmt"

func main() {
	fmt.Println(double(5))

	fmt.Println(triple(4)) // "12"
}

func double(x int) (res int) {
	defer func() {
		fmt.Printf("\ndouble %d\n", res)
	}()
	return x * 2
}

func triple(x int) (result int) {
	defer func() { result += x }()
	return double(x)
}

package main

import (
	"fmt"
	"runtime"
)

func main() {
	a()
}

func a() {
	fmt.Println("a0")
	fmt.Println(runtime.Caller(0))
	fmt.Println("a1")
	fmt.Println(runtime.Caller(1))
	b()
}

func b() {
	fmt.Println("b0")
	fmt.Println(runtime.Caller(0))
	fmt.Println("b1")
	fmt.Println(runtime.Caller(1))
	fmt.Println("b2")
	fmt.Println(runtime.Caller(2))
}

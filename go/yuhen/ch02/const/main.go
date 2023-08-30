// main is const test
package main

import (
	"fmt"
	"unsafe"
)

func main() {

	const (
		x uint16 = 120
		y
		s = "abc"
		z
	)

	fmt.Printf("%T %[1]v\n", y)

	fmt.Printf("%T %[1]v\n", z)

	const (
		psrSize = unsafe.Sizeof(uintptr(0))
		strSize = len("hello,world!")
	)
	fmt.Println(psrSize)
	fmt.Println(strSize)

	const (
		Sunday = iota
		Monday
		Hi = 100
		Wen
	)
	fmt.Println(Sunday, Monday, Hi, Wen)
}

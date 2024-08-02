package main

import (
	"flag"
	"fmt"
)

func main() {
	x := 1
	p := &x         // p 的类型为 *int 指向变量 x
	fmt.Println(*p) // 1

	*p = 2         // 与 x = 2 等价
	fmt.Println(x) // 2

	// 相等比较
	var p1 *int
	var p2 *int
	// var p3 *float32

	fmt.Println(p1, p2, &p1, &p2, p1 == p2) // <nil> <nil> 0xc000058028 0xc000058030 true
	// fmt.Println(*p1 + *p2)                  // panic: runtime error: invalid memory address or nil pointer dereference

	// fmt.Println(p1 == p3) // invalid operation: p1 == p3 (mismatched types *int and *float32)
	x = 1
	p1 = &x
	p2 = &x
	fmt.Println(p1, p2, &p1, &p2, p1 == p2) // 0xc0000120d0 0xc0000120d0 0xc000058028 0xc000058030 true

	fmt.Println(f() == f())

	fmt.Println(*n)
	flag.Parse()
	fmt.Println(*n)
}

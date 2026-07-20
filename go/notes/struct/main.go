package main

import (
	"fmt"
	"unsafe"
)

type Bad struct {
	a bool
	b int64
	c bool
}

type Good struct {
	a, c bool
	b    int64
}

func main() {

	b := Bad{}
	fmt.Println()

	fmt.Printf("bad.a Offsetof=%d Sizeof = %d Alignof=%d\n", unsafe.Offsetof(b.a), unsafe.Sizeof(b.a), unsafe.Alignof(b.a))
	fmt.Printf("bad.b Offsetof=%d Sizeof = %d Alignof=%d\n", unsafe.Offsetof(b.b), unsafe.Sizeof(b.b), unsafe.Alignof(b.b))
	fmt.Printf("bad.c Offsetof=%d Sizeof = %d Alignof=%d\n", unsafe.Offsetof(b.c), unsafe.Sizeof(b.c), unsafe.Alignof(b.c))

	fmt.Printf("bad Sizeof = %d\n", unsafe.Sizeof(b))

	g := Good{}
	fmt.Println()

	fmt.Printf("good.b Offsetof=%d Sizeof = %d Alignof=%d\n", unsafe.Offsetof(g.b), unsafe.Sizeof(g.b), unsafe.Alignof(g.b))
	fmt.Printf("good.a Offsetof=%d Sizeof = %d Alignof=%d\n", unsafe.Offsetof(g.a), unsafe.Sizeof(g.a), unsafe.Alignof(g.a))
	fmt.Printf("good.c Offsetof=%d Sizeof = %d Alignof=%d\n", unsafe.Offsetof(g.c), unsafe.Sizeof(g.c), unsafe.Alignof(g.c))

	fmt.Printf("good Sizeof = %d\n", unsafe.Sizeof(g))

	type T struct{ V int }
	s := []T{{1}, {2}, {3}}

	var ptrs []*T
	for _, v := range s {
		ptrs = append(ptrs, &v) // 错误：&v 是循环变量的地址
	}

	for _, v := range ptrs {
		v.V = 5
	}

	for i, v := range ptrs {
		fmt.Println(i, v)
	}

	for i, v := range s {
		fmt.Println(i, v)
	}
}

package main

import "fmt"

func main() {
	x, y := 1, 2
	x, y = y+3, x+2
	println(x, y)
	f1 := f()
	f2 := f()
	fmt.Println(f1 == f2, f1, f2)
	v := 1
	incr(&v)
	fmt.Println(incr(&v))

}

func f() *int {
	v := 1
	return &v
}

func incr(p *int) int {
	*p++
	return *p
}

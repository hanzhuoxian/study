package main

import (
	"container/list"
	"fmt"
)

var aa = cc
var cc = 1

func main() {
	l := list.New()
	e := l.PushBack(3)
	fmt.Println(l.Len())

	a := l.Remove(e)
	a1 := a.(int)
	fmt.Printf("%d, %[1]T %d\n", a, a1+3)
}

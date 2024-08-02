package main

import (
	"container/list"
	"fmt"
)

func main() {
	lt := New()
	fmt.Printf("%#v", lt)

	l1 := list.New()
	e := l1.PushBack(3)
	l1.Init()
	fmt.Println(e)
	l1.Remove(e)
}

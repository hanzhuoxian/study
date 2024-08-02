package main

import (
	"fmt"
)

func main() {
	var x int
	p := &x
	fmt.Println(*p)
	*p = 1
	fmt.Println(*p)
}

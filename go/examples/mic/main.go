package main

import (
	"fmt"
	"sync/atomic"
)

func main() {
	x := new(int64)
	atomic.AddInt64(x, 1)
	fmt.Println(*x)

	atomic.CompareAndSwapInt64(x, 2, 2)
	fmt.Println(*x)
}

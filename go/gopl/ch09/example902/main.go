package main

import (
	"fmt"
	"sync"
)

// **练习 9.2：** 重写2.6.2节中的PopCount的例子，使用sync.Once，只在第一次需要用到的时候进行初始化。
// （虽然实际上，对PopCount这样很小且高度优化的函数进行同步可能代价没法接受。）

var pc [256]byte
var syncOnce sync.Once

func PopCount(x uint64) int {
	syncOnce.Do(func() {
		for i := range pc {
			pc[i] = pc[i/2] + byte(i&1)
		}
	})

	num := 0
	for i := range 8 {
		num += int(pc[byte(x>>(i*8))])
	}
	return num
}

func main() {
	fmt.Println(PopCount(2))
}

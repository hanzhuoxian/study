package main

import "fmt"

// **练习5.19：** 使用panic和recover编写一个不包含return语句但能返回一个非零值的函数。

func main() {
	fmt.Print(f())
}

func f() (res int) {
	defer func() {
		if r := recover(); r != nil {
			res = 1
		}
	}()
	panic("panic")
}

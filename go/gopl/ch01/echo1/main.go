package main

// echo1 打印命令行的参数

import (
	"fmt"
	"os"
)

func main() {
	// go 在声明变量时没有显示初始化，go 语言会对变量隐式赋 0 值，字符串会赋值空字符串
	var s, sep string
	// 经典的for循环
	for i := 1; i <= len(os.Args[1:]); i++ {
		// += 是一个赋值运算符
		s += sep + os.Args[i]
		sep = " "
	}
	fmt.Println(s)
}

package main

//打印命令行参数的索引和值

import (
	"fmt"
	"os"
)

func main() {
	for index, value := range os.Args[1:] {
		fmt.Println(index, value)
	}
}

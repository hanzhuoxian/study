package main

import (
	"bufio"
	"fmt"
	"os"
)

// `dup` 的第一个版本打印标准输入中多次出现的行，以重复次数开头。
// 该程序将引入 `if` 语句，`map` 数据类型以及 `bufio` 包。

func main() {
	count := make(map[string]int)
	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		count[input.Text()]++
		if input.Err() != nil {
			panic("input is error")
		}
	}

	for line, n := range count {
		if n > 1 {
			fmt.Printf("%d\t%s\n", n, line)
		}
	}
}

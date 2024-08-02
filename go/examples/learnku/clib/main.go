package main

// 编译源码里必须有main package

import (
	"C" // 源码必须import "C"

	"github.com/spf13/cast"
)

// 导出符号位于main package，且前一行有//export NAME，注意// 与  export 中间不能有空格
//
//export myatoi
func myatoi(s string) int {
	return cast.ToInt(s)
}

func main() {}

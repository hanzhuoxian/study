package main

import (
	"fmt"
	"unicode/utf8"
)

// 练习 4.7： 修改reverse函数用于原地反转UTF-8编码的[]byte。是否可以不用分配额外的内存？

func main() {
	u := []byte("!dlrow olleh")
	u = reverse(u)
	fmt.Println(string(u))
}

func reverse(u []byte) []byte {
	if len(u) <= 1 {
		return u
	}

	// 第一步：原地反转每个多字节 rune 内部的字节
	for s := 0; s < len(u); {
		_, size := utf8.DecodeRune(u[s:])
		reverseBytes(u[s : s+size])
		s += size
	}
	// 第二步：原地反转整个 slice
	reverseBytes(u)
	return u
}

// reverseBytes 原地反转字节序列，不分配额外内存
func reverseBytes(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}

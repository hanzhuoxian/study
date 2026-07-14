package main

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// 练习 4.6： 编写一个函数，原地将一个UTF-8编码的[]byte类型的slice中相邻的空格（参考unicode.IsSpace）替换成一个空格返回

func main() {
	cases := []string{
		"Hello 	 	 	 	world  	 	 	 	!",
		"a   b   c",
		"a   b   c   d",
		"x    y    z",
		"中文　　空格　合并", // 全角空格
		"  leading and trailing  ",
		"nospace",
		"",
	}
	for _, c := range cases {
		fmt.Printf("%q -> %q\n", c, string(MergeSpace([]byte(c))))
	}
}

// MergeSpace 原地把相邻空格合并成一个 ASCII 空格
func MergeSpace(u []byte) []byte {
	w := 0
	prevSpace := false
	for r := 0; r < len(u); {
		c, size := utf8.DecodeRune(u[r:])
		if unicode.IsSpace(c) {
			if !prevSpace {
				u[w] = ' '
				w++
				prevSpace = true
			}
		} else {
			copy(u[w:], u[r:r+size])
			w += size
			prevSpace = false
		}
		r += size
	}
	return u[:w]
}

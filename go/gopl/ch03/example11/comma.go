package main

import (
	"bytes"
	"fmt"
)

func comma(s string) string {
	n := len(s)
	symbol := false
	if s[0] == '+' || s[0] == '-' {
		symbol = true
	}
	if symbol && n <= 4 || !symbol && n <= 3 {
		return s
	}

	var buf bytes.Buffer
	j := n % 3
	if j == 0 {
		j = 3
	}
	buf.Write([]byte(s[:j]))
	for i := j; i < n; i += 3 {
		buf.Write([]byte(","))
		buf.Write([]byte(s[i : i+3]))
	}
	return buf.String()
}

func main() {
	fmt.Println(comma("123456"))
	fmt.Println(comma("123"))
}

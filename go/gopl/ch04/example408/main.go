package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"unicode"
	"unicode/utf8"
)

// 练习 4.8： 修改charcount程序，使用unicode.IsLetter等相关的函数，统计字母、数字等Unicode中不同的字符类别。
const (
	Letter = iota
	Digit
	Space
	Punct
	Symbol
	Control
	Mark
	Other
)

// typeNames 用于把类别常量映射为可读名称。
var typeNames = [...]string{
	Letter:  "letter",
	Digit:   "digit",
	Space:   "space",
	Punct:   "punct",
	Symbol:  "symbol",
	Control: "control",
	Mark:    "mark",
	Other:   "other",
}

func main() {
	counts := make(map[int]int)     // 每个类别的字符数量
	var utfLen [utf8.UTFMax + 1]int // 各 UTF-8 编码长度的字符数量
	invalid := 0                    // 非法 UTF-8 字符数量

	in := bufio.NewReader(os.Stdin)
	for {
		r, s, err := in.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "example8 read failed: %v\n", err)
			os.Exit(1)
		}

		if r == unicode.ReplacementChar && s == 1 {
			invalid++
			continue
		}
		counts[RuneType(r)]++
		utfLen[s]++
	}

	fmt.Printf("type\tcount\n")
	for t, n := range counts {
		fmt.Printf("%s\t%d\n", typeNames[t], n)
	}

	fmt.Printf("\nlen\tcount\n")
	for i, n := range utfLen[1:] {
		fmt.Printf("%d\t%d\n", i+1, n)
	}
	if invalid > 0 {
		fmt.Printf("\n%d invalid UTF-8 characters\n", invalid)
	}
}

func RuneType(r rune) int {
	switch {
	case unicode.IsLetter(r):
		return Letter
	case unicode.IsNumber(r):
		return Digit
	case unicode.IsSpace(r):
		return Space
	case unicode.IsPunct(r):
		return Punct
	case unicode.IsSymbol(r):
		return Symbol
	case unicode.IsControl(r):
		return Control
	case unicode.IsMark(r):
		return Mark
	default:
		return Other
	}
}

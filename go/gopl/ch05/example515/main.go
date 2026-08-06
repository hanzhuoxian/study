package main

import "fmt"

// **练习5.15：** 编写类似sum的可变参数函数max和min。考虑不传参时，max和min该如何处理，再编写至少接收1个参数的版本。
func main() {
	fmt.Println(max(3, 1, 4, 1, 5, 9))
	fmt.Println(min(3, 1, 4, 1, 5, 9))

	_, ok := maxOrZero()
	fmt.Println(ok)
}

// maxOrZero 是可以不传参的版本：没有参数时无法给出有意义的最大值，
// 因此用 ok 标识结果是否有效，而不是返回一个容易被误用的哨兵值。
func maxOrZero(vals ...int) (max int, ok bool) {
	if len(vals) == 0 {
		return 0, false
	}
	max = vals[0]
	for _, v := range vals[1:] {
		if v > max {
			max = v
		}
	}
	return max, true
}

// max 要求至少传入一个参数，通过把第一个参数单独列出，在编译期就排除了空调用。
func max(first int, rest ...int) int {
	m := first
	for _, v := range rest {
		if v > m {
			m = v
		}
	}
	return m
}

// min 同 max，至少接收一个参数。
func min(first int, rest ...int) int {
	m := first
	for _, v := range rest {
		if v < m {
			m = v
		}
	}
	return m
}

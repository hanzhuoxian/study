package main

import "fmt"

// 练习 4.5： 写一个函数在原地完成消除[]string中相邻重复的字符串的操作。
func main() {
	s := []string{"hello", "h", "h", "j", "j", "hello", "hello", "hello"}
	s = uniq(s)
	fmt.Println(s)
}

func uniq(s []string) []string {
	if len(s) <= 0 {
		return s
	}
	last := 1
	for i, v := range s {
		if i == 0 || s[last-1] == v {
			continue
		}
		s[last] = v
		last++
	}

	return s[:last]
}

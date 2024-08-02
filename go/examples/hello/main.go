package main

import (
	"fmt"
	"regexp"
)

func main() {
	fmt.Println("Hello​World​!")
	fmt.Println(removeInvisibleChars("Hello​Wo r l d​!"))
}

func removeInvisibleChars(input string) string {
	// 使用正则表达式匹配不可见字符
	re := regexp.MustCompile(`\p{C}`)

	// 使用正则表达式替换不可见字符为空字符串
	filteredText := re.ReplaceAllString(input, "")
	return filteredText
}

package main

import "fmt"

func main() {
	fmt.Println("hello")
}

// Btoi bool 类型转换为 int
func Btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Itob int 类型转换为 bool
func Itob(i int) bool {
	return i != 0
}

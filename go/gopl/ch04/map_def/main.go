package main

import "fmt"

func main() {
	var m map[int]int
	fmt.Println(m == nil)
	fmt.Println(len(m) == 0)

	m1 := map[int]int{
		1: 1,
	}
	fmt.Println(m1)

	if i, ok := m1[1]; ok {
		fmt.Println(i)
	}
	if i, ok := m1[2]; !ok {
		fmt.Println("no", i)
	}
}

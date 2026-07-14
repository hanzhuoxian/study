package main

import "fmt"

func main() {
	s := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for i, v := range s {
		fmt.Printf("v: %x, s[%d]: %x\n", &v, i, &s[i])
		v *= 2
	}
	fmt.Println(s)

	s = append(s, s...)
	fmt.Println(s)

	slice := append([]byte("hello "), "world"...)
	fmt.Println(string(slice))
}

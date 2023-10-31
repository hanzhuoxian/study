package main

import (
	"fmt"
	mrand "math/rand"
)

type StringSlice []string

func (s StringSlice) Len() int           { return len(s) }
func (s StringSlice) Less(i, j int) bool { return s[i] < s[j] }
func (s StringSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func main() {
	var x interface{}
	x = 1
	switch x := x.(type) {
	case int:
		fmt.Printf("%T %v\n", x, x)
	default:
		fmt.Printf("%T %v", x, x)
	}
	mrand.Float32()

}

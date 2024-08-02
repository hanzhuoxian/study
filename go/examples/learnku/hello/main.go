package main

import (
	"fmt"
)

const siteName = "learnku"

type Porint struct{ X, Y int }

func main() {
	for i := 0; i < 5; i++ {
		for j := 0; i < 5; j++ {
			if j == 3 {
				goto L
			}
		}
	}
L:
	fmt.Println("L")
}

func add(x, y int) int {
	return x + y
}

type Celsius float64

func (c Celsius) String() string {
	return fmt.Sprintf("%g° C", c)
}

package main

import "fmt"

func main() {
	for i := 0; i < 5; i++ {
		fmt.Println(i)
	}
}

// GODEBUG=schedtrace=1000 go run .
